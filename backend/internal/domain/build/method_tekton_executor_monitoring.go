package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (e *MethodTektonExecutor) monitorPipelineRun(ctx context.Context, executionID uuid.UUID, pipelineRun *tektonv1.PipelineRun, k8sClient kubernetes.Interface, tektonClient tektonclient.Interface) {
	defer func() {
		e.mu.Lock()
		delete(e.running, executionID.String())
		e.mu.Unlock()
		_ = e.service.ReleaseMonitoringLease(context.Background(), executionID, e.instanceID)
	}()

	pipelineRunName := pipelineRun.Name
	namespace := pipelineRun.Namespace

	e.logger.Info("Monitoring PipelineRun",
		zap.String("executionID", executionID.String()),
		zap.String("name", pipelineRunName),
		zap.String("namespace", namespace))

	ticker := time.NewTicker(e.pipelineRunPollInterval())
	defer ticker.Stop()
	leaseTicker := time.NewTicker(e.leaseRenewEvery)
	defer leaseTicker.Stop()

	timeout := 2 * time.Hour
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()
	logCursor := make(map[string]int)
	logWarned := make(map[string]bool)
	stepProgress := make(map[string]string)
	pipelineProgress := make(map[string]string)

	for {
		select {
		case <-ctx.Done():
			e.finalizeExecution(ctx, executionID, ExecutionCancelled, "Execution cancelled", nil)
			return
		case <-timeoutTimer.C:
			e.finalizeExecution(ctx, executionID, ExecutionFailed, fmt.Sprintf("PipelineRun timed out after %v", timeout), nil)
			return
		case <-leaseTicker.C:
			renewed, err := e.service.RenewMonitoringLease(ctx, executionID, e.instanceID, e.monitorLeaseTTl)
			if err != nil {
				e.logger.Warn("Failed to renew monitoring lease", zap.String("execution_id", executionID.String()), zap.Error(err))
				return
			}
			if !renewed {
				e.logger.Info("Monitoring lease lost; stopping monitor loop", zap.String("execution_id", executionID.String()))
				return
			}
		case <-ticker.C:
			e.capturePipelineRunTaskLogs(ctx, executionID, namespace, pipelineRunName, k8sClient, tektonClient, logCursor, logWarned, stepProgress)

			pr, err := tektonClient.TektonV1().PipelineRuns(namespace).Get(ctx, pipelineRunName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					e.finalizeExecution(ctx, executionID, ExecutionCancelled, fmt.Sprintf("PipelineRun %s/%s not found", namespace, pipelineRunName), nil)
					return
				}
				e.logger.Error("Failed to get PipelineRun status", zap.Error(err))
				continue
			}

			if pr.Status.CompletionTime != nil {
				e.capturePipelineRunTaskLogs(ctx, executionID, namespace, pipelineRunName, k8sClient, tektonClient, logCursor, logWarned, stepProgress)
				failureMessage := ""
				if len(pr.Status.Conditions) > 0 && pr.Status.Conditions[0].Status == "False" {
					if summary := e.summarizePipelineRunFailure(ctx, namespace, pipelineRunName, tektonClient); summary != nil {
						failureMessage = fmt.Sprintf("Task %s failed at stage %s", summary.TaskRun, summary.Stage)
						if summary.Reason != "" {
							failureMessage += fmt.Sprintf(": %s", summary.Reason)
						}
						if summary.Message != "" {
							failureMessage += fmt.Sprintf(" - %s", summary.Message)
						}

						meta := map[string]interface{}{
							"source":       "tekton",
							"pipeline_run": pipelineRunName,
							"task_run":     summary.TaskRun,
							"stage":        summary.Stage,
							"failure":      true,
							"error_type":   "task_failed",
						}
						if summary.PipelineTask != "" {
							meta["pipeline_task"] = summary.PipelineTask
						}
						if summary.Step != "" {
							meta["step"] = summary.Step
						}
						if summary.Pod != "" {
							meta["pod"] = summary.Pod
						}
						if summary.Reason != "" {
							meta["reason"] = summary.Reason
						}
						raw, _ := json.Marshal(meta)
						_ = e.service.AddLog(ctx, executionID, LogError, failureMessage, raw)
					}
				}
				e.finalizeExecutionFromPipelineRun(ctx, executionID, pr, failureMessage, tektonClient)
				return
			}

			e.logPipelineRunWaitingProgress(ctx, executionID, pipelineRunName, namespace, pr, tektonClient, pipelineProgress)

			if len(pr.Status.Conditions) > 0 {
				e.logger.Debug("PipelineRun status",
					zap.String("name", pipelineRunName),
					zap.String("status", string(pr.Status.Conditions[0].Status)))
			}
		}
	}
}

func (e *MethodTektonExecutor) logPipelineRunWaitingProgress(
	ctx context.Context,
	executionID uuid.UUID,
	pipelineRunName string,
	namespace string,
	pr *tektonv1.PipelineRun,
	tektonClient tektonclient.Interface,
	progress map[string]string,
) {
	if pr == nil || progress == nil {
		return
	}
	if pr.Status.CompletionTime != nil {
		return
	}

	waiting := e.describeWaitingPipelineTasks(ctx, namespace, pr, tektonClient)
	snapshot := strings.Join(waiting, "|")
	key := "__pipeline_waiting__"
	if previous, exists := progress[key]; exists && previous == snapshot {
		return
	}
	progress[key] = snapshot
	if snapshot == "" {
		return
	}

	message := fmt.Sprintf("Pipeline waiting on next stage: %s", strings.Join(waiting, "; "))
	metadata, _ := json.Marshal(map[string]interface{}{
		"source":        "tekton",
		"phase":         "pipeline_waiting",
		"pipeline_run":  pipelineRunName,
		"waiting_tasks": waiting,
	})
	_ = e.service.AddLog(ctx, executionID, LogInfo, message, metadata)
}

func (e *MethodTektonExecutor) describeWaitingPipelineTasks(
	ctx context.Context,
	namespace string,
	pr *tektonv1.PipelineRun,
	tektonClient tektonclient.Interface,
) []string {
	taskStates, err := e.collectPipelineTaskStates(ctx, namespace, pr, tektonClient)
	if err != nil {
		e.logger.Debug("Failed to collect task states for pipeline waiting progress",
			zap.String("namespace", namespace),
			zap.String("pipeline_run", pr.Name),
			zap.Error(err))
		return nil
	}
	if len(taskStates) == 0 {
		return nil
	}

	deps := collectPipelineTaskDependencies(pr)
	names := make([]string, 0, len(taskStates))
	for name := range taskStates {
		names = append(names, name)
	}
	sort.Strings(names)

	waiting := make([]string, 0)
	for _, name := range names {
		if taskStates[name] != "pending" {
			continue
		}
		blockers := make([]string, 0)
		for _, dep := range deps[name] {
			state, ok := taskStates[dep]
			if !ok || (state != "completed" && state != "skipped") {
				blockers = append(blockers, dep)
			}
		}
		if len(blockers) > 0 {
			waiting = append(waiting, fmt.Sprintf("%s (waiting on %s)", name, strings.Join(blockers, ",")))
			continue
		}
		waiting = append(waiting, fmt.Sprintf("%s (pending scheduling)", name))
	}

	const maxWaitingItems = 3
	if len(waiting) > maxWaitingItems {
		remaining := len(waiting) - maxWaitingItems
		waiting = append(waiting[:maxWaitingItems], fmt.Sprintf("+%d more pending task(s)", remaining))
	}
	return waiting
}

func (e *MethodTektonExecutor) collectPipelineTaskStates(
	ctx context.Context,
	namespace string,
	pr *tektonv1.PipelineRun,
	tektonClient tektonclient.Interface,
) (map[string]string, error) {
	states := make(map[string]string)

	pipelineSpec := resolvePipelineRunSpec(pr)
	if pipelineSpec != nil {
		for _, task := range pipelineSpec.Tasks {
			name := strings.TrimSpace(task.Name)
			if name == "" {
				continue
			}
			states[name] = "pending"
		}
	}

	if pr != nil {
		for _, skipped := range pr.Status.SkippedTasks {
			name := strings.TrimSpace(skipped.Name)
			if name == "" {
				continue
			}
			states[name] = "skipped"
		}
		for _, child := range pr.Status.ChildReferences {
			name := strings.TrimSpace(child.PipelineTaskName)
			if name == "" {
				continue
			}
			if _, exists := states[name]; !exists {
				states[name] = "pending"
			}
		}
	}

	if tektonClient == nil || pr == nil || strings.TrimSpace(pr.Name) == "" {
		return states, nil
	}

	taskRuns, err := tektonClient.TektonV1().TaskRuns(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", pr.Name),
	})
	if err != nil {
		return states, err
	}
	for _, tr := range taskRuns.Items {
		name := strings.TrimSpace(tr.Labels["tekton.dev/pipelineTask"])
		if name == "" {
			name = strings.TrimSpace(tr.Name)
		}
		if name == "" {
			continue
		}
		states[name] = pipelineTaskStateFromTaskRun(tr)
	}

	return states, nil
}

func resolvePipelineRunSpec(pr *tektonv1.PipelineRun) *tektonv1.PipelineSpec {
	if pr == nil {
		return nil
	}
	if pr.Status.PipelineSpec != nil {
		return pr.Status.PipelineSpec
	}
	return pr.Spec.PipelineSpec
}

func collectPipelineTaskDependencies(pr *tektonv1.PipelineRun) map[string][]string {
	deps := make(map[string][]string)
	pipelineSpec := resolvePipelineRunSpec(pr)
	if pipelineSpec == nil {
		return deps
	}

	for _, task := range pipelineSpec.Tasks {
		name := strings.TrimSpace(task.Name)
		if name == "" {
			continue
		}
		if len(task.RunAfter) == 0 {
			deps[name] = nil
			continue
		}
		runAfter := make([]string, 0, len(task.RunAfter))
		for _, dep := range task.RunAfter {
			trimmed := strings.TrimSpace(dep)
			if trimmed == "" {
				continue
			}
			runAfter = append(runAfter, trimmed)
		}
		deps[name] = runAfter
	}

	return deps
}

func pipelineTaskStateFromTaskRun(tr tektonv1.TaskRun) string {
	for _, condition := range tr.Status.Conditions {
		if !strings.EqualFold(string(condition.Type), "Succeeded") {
			continue
		}
		if condition.Status == "True" {
			return "completed"
		}
		if condition.Status == "False" {
			return "failed"
		}
	}
	return "running"
}

func (e *MethodTektonExecutor) pipelineRunPollInterval() time.Duration {
	interval := 5 * time.Second

	tektonMonitorModeMu.RLock()
	override := tektonMonitorEventDriven
	tektonMonitorModeMu.RUnlock()
	if override != nil {
		if *override {
			return 15 * time.Second
		}
		return interval
	}

	value := strings.TrimSpace(strings.ToLower(os.Getenv("IF_BUILD_MONITOR_EVENT_DRIVEN_ENABLED")))
	switch value {
	case "1", "true", "yes", "y", "on":
		return 15 * time.Second
	default:
		return interval
	}
}

func (e *MethodTektonExecutor) capturePipelineRunTaskLogs(
	ctx context.Context,
	executionID uuid.UUID,
	namespace string,
	pipelineRunName string,
	k8sClient kubernetes.Interface,
	tektonClient tektonclient.Interface,
	cursor map[string]int,
	warned map[string]bool,
	progress map[string]string,
) {
	if tektonClient == nil || k8sClient == nil {
		return
	}

	taskRuns, err := tektonClient.TektonV1().TaskRuns(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", pipelineRunName),
	})
	if err != nil {
		if e != nil {
			e.logger.Debug("Failed to list task runs for log capture",
				zap.String("execution_id", executionID.String()),
				zap.String("pipeline_run", pipelineRunName),
				zap.Error(err))
		}
		if !warned["taskrun_list"] {
			meta, _ := json.Marshal(map[string]interface{}{
				"source":       "tekton",
				"pipeline_run": pipelineRunName,
				"error_type":   "taskrun_list_failed",
			})
			_ = e.service.AddLog(ctx, executionID, LogWarn, fmt.Sprintf("Unable to list TaskRuns for Tekton log capture: %v", err), meta)
			warned["taskrun_list"] = true
		}
		return
	}
	if len(taskRuns.Items) == 0 {
		return
	}

	sort.SliceStable(taskRuns.Items, func(i, j int) bool {
		left := taskRuns.Items[i]
		right := taskRuns.Items[j]
		if !left.CreationTimestamp.Equal(&right.CreationTimestamp) {
			return left.CreationTimestamp.Before(&right.CreationTimestamp)
		}
		return left.Name < right.Name
	})

	for _, taskRun := range taskRuns.Items {
		podName := strings.TrimSpace(taskRun.Status.PodName)
		if podName == "" {
			podName = resolveTaskRunPodName(ctx, k8sClient, namespace, taskRun.Name, e.logger)
		}
		if podName == "" {
			continue
		}

		containerNames := make([]string, 0)
		seenContainers := make(map[string]struct{})
		for _, step := range taskRun.Status.Steps {
			containerName := strings.TrimSpace(step.Container)
			if containerName == "" {
				containerName = strings.TrimSpace(step.Name)
			}
			if containerName == "" {
				continue
			}
			if _, exists := seenContainers[containerName]; exists {
				continue
			}
			seenContainers[containerName] = struct{}{}
			containerNames = append(containerNames, containerName)
			e.logTektonStepProgress(ctx, executionID, pipelineRunName, taskRun.Name, podName, containerName, step, progress)
		}

		if len(containerNames) == 0 {
			pod, podErr := k8sClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if podErr == nil && pod != nil {
				for _, c := range pod.Spec.Containers {
					containerName := strings.TrimSpace(c.Name)
					if containerName == "" {
						continue
					}
					if _, exists := seenContainers[containerName]; exists {
						continue
					}
					seenContainers[containerName] = struct{}{}
					containerNames = append(containerNames, containerName)
				}
			}
		}

		for _, containerName := range containerNames {
			req := k8sClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
				Container:  containerName,
				Timestamps: true,
			})
			stream, err := req.Stream(ctx)
			if err != nil {
				if isTransientTektonLogStreamError(err) {
					if e != nil {
						e.logger.Debug("Tekton step log stream not ready yet",
							zap.String("execution_id", executionID.String()),
							zap.String("pipeline_run", pipelineRunName),
							zap.String("task_run", taskRun.Name),
							zap.String("step", containerName),
							zap.String("pod", podName),
							zap.Error(err))
					}
					continue
				}
				warnKey := "stream_error:" + taskRun.Name + "/" + containerName
				if !warned[warnKey] {
					meta, _ := json.Marshal(map[string]interface{}{
						"source":       "tekton",
						"pipeline_run": pipelineRunName,
						"task_run":     taskRun.Name,
						"step":         containerName,
						"pod":          podName,
						"error_type":   "pod_log_stream_failed",
					})
					_ = e.service.AddLog(ctx, executionID, LogWarn, fmt.Sprintf("Unable to stream Tekton step logs for %s/%s: %v", taskRun.Name, containerName, err), meta)
					warned[warnKey] = true
				}
				continue
			}
			payload, readErr := io.ReadAll(stream)
			_ = stream.Close()
			if readErr != nil || len(payload) == 0 {
				continue
			}

			key := taskRun.Name + "/" + containerName
			lines := splitLogLines(string(payload))
			start := cursor[key]
			if start < 0 {
				start = 0
			}
			if start > len(lines) {
				start = 0
			}
			for _, line := range lines[start:] {
				raw := strings.TrimSpace(line)
				if raw == "" {
					continue
				}
				timestamp, content := parseTimestampedLogLine(raw)
				message := fmt.Sprintf("[%s/%s] %s", taskRun.Name, containerName, content)
				level := logLevelFromTektonLine(content)
				metadata, _ := json.Marshal(map[string]interface{}{
					"source":       "tekton",
					"pipeline_run": pipelineRunName,
					"task_run":     taskRun.Name,
					"step":         containerName,
					"pod":          podName,
					"timestamp":    timestamp,
				})
				_ = e.service.AddLog(ctx, executionID, level, message, metadata)
			}
			cursor[key] = len(lines)
		}
	}
}

func (e *MethodTektonExecutor) logTektonStepProgress(
	ctx context.Context,
	executionID uuid.UUID,
	pipelineRunName string,
	taskRunName string,
	podName string,
	containerName string,
	step tektonv1.StepState,
	progress map[string]string,
) {
	if progress == nil {
		return
	}

	state, reason, message, level := tektonStepProgressState(step)
	if state == "" {
		return
	}

	key := taskRunName + "/" + containerName
	if previous, exists := progress[key]; exists && previous == state {
		return
	}
	progress[key] = state

	detail := ""
	if reason != "" {
		detail = " (" + reason + ")"
	}
	text := fmt.Sprintf("Tekton step %s: %s/%s%s", state, taskRunName, containerName, detail)
	if message != "" && state == "failed" {
		text = fmt.Sprintf("%s - %s", text, strings.TrimSpace(message))
	}

	metadata, _ := json.Marshal(map[string]interface{}{
		"source":       "tekton",
		"phase":        "progress",
		"pipeline_run": pipelineRunName,
		"task_run":     taskRunName,
		"step":         containerName,
		"pod":          podName,
		"state":        state,
		"reason":       reason,
	})
	_ = e.service.AddLog(ctx, executionID, level, text, metadata)
}

func tektonStepProgressState(step tektonv1.StepState) (state string, reason string, message string, level LogLevel) {
	if step.Waiting != nil {
		waitingReason := strings.TrimSpace(step.Waiting.Reason)
		if waitingReason == "" {
			waitingReason = "waiting"
		}
		return "waiting", waitingReason, strings.TrimSpace(step.Waiting.Message), LogInfo
	}
	if step.Running != nil {
		return "running", "", "", LogInfo
	}
	if step.Terminated != nil {
		termReason := strings.TrimSpace(step.Terminated.Reason)
		if termReason == "" {
			termReason = fmt.Sprintf("exit_code=%d", step.Terminated.ExitCode)
		}
		if step.Terminated.ExitCode == 0 {
			return "completed", termReason, strings.TrimSpace(step.Terminated.Message), LogInfo
		}
		return "failed", termReason, strings.TrimSpace(step.Terminated.Message), LogError
	}
	return "", "", "", LogInfo
}

func isTransientTektonLogStreamError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "podinitializing") ||
		strings.Contains(msg, "containercreating") ||
		strings.Contains(msg, "is waiting to start")
}

func resolveTaskRunPodName(ctx context.Context, k8sClient kubernetes.Interface, namespace, taskRunName string, logger *zap.Logger) string {
	if k8sClient == nil || namespace == "" || taskRunName == "" {
		return ""
	}

	pods, err := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/taskRun=%s", taskRunName),
	})
	if err != nil {
		if logger != nil {
			logger.Debug("Failed to list pods for TaskRun fallback lookup",
				zap.String("namespace", namespace),
				zap.String("task_run", taskRunName),
				zap.Error(err))
		}
		return ""
	}
	if len(pods.Items) == 0 {
		return ""
	}

	best := pods.Items[0]
	for i := 1; i < len(pods.Items); i++ {
		candidate := pods.Items[i]
		if podPhaseRank(candidate.Status.Phase) > podPhaseRank(best.Status.Phase) {
			best = candidate
			continue
		}
		if podPhaseRank(candidate.Status.Phase) < podPhaseRank(best.Status.Phase) {
			continue
		}
		if candidate.CreationTimestamp.After(best.CreationTimestamp.Time) {
			best = candidate
		}
	}
	return strings.TrimSpace(best.Name)
}

func podPhaseRank(phase corev1.PodPhase) int {
	switch phase {
	case corev1.PodRunning:
		return 4
	case corev1.PodPending:
		return 3
	case corev1.PodSucceeded:
		return 2
	case corev1.PodFailed:
		return 1
	default:
		return 0
	}
}

func splitLogLines(payload string) []string {
	normalized := strings.ReplaceAll(payload, "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	if strings.TrimSpace(normalized) == "" {
		return nil
	}
	return strings.Split(normalized, "\n")
}

func parseTimestampedLogLine(line string) (string, string) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return "", line
	}
	if _, err := time.Parse(time.RFC3339Nano, parts[0]); err != nil {
		return "", line
	}
	return parts[0], parts[1]
}

func logLevelFromTektonLine(content string) LogLevel {
	s := strings.TrimSpace(strings.ToLower(content))
	if s == "" {
		return LogInfo
	}

	if strings.Contains(s, " panic") || strings.HasPrefix(s, "panic:") {
		return LogError
	}
	if strings.Contains(s, " fatal") || strings.HasPrefix(s, "fatal:") {
		return LogError
	}
	if strings.Contains(s, " error") || strings.HasPrefix(s, "error:") {
		return LogError
	}
	if strings.Contains(s, " warn") || strings.HasPrefix(s, "warn:") || strings.Contains(s, " warning") {
		return LogWarn
	}
	return LogInfo
}

type tektonFailureSummary struct {
	PipelineTask string
	TaskRun      string
	Step         string
	Pod          string
	Stage        string
	Reason       string
	Message      string
}

func inferExecutionStage(taskRunName, stepName string) string {
	signal := strings.ToLower(strings.TrimSpace(taskRunName + " " + stepName))
	switch {
	case strings.Contains(signal, "sbom"):
		return "sbom"
	case strings.Contains(signal, "scan"):
		return "scan"
	case strings.Contains(signal, "push"):
		return "publish"
	case strings.Contains(signal, "build"):
		return "build"
	case strings.Contains(signal, "clone"):
		return "dispatched"
	default:
		return "build"
	}
}

func (e *MethodTektonExecutor) summarizePipelineRunFailure(
	ctx context.Context,
	namespace string,
	pipelineRunName string,
	tektonClient tektonclient.Interface,
) *tektonFailureSummary {
	if tektonClient == nil {
		return nil
	}

	taskRuns, err := tektonClient.TektonV1().TaskRuns(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", pipelineRunName),
	})
	if err != nil || len(taskRuns.Items) == 0 {
		return nil
	}

	sort.SliceStable(taskRuns.Items, func(i, j int) bool {
		left := taskRuns.Items[i]
		right := taskRuns.Items[j]
		if !left.CreationTimestamp.Equal(&right.CreationTimestamp) {
			return left.CreationTimestamp.Before(&right.CreationTimestamp)
		}
		return left.Name < right.Name
	})

	for _, tr := range taskRuns.Items {
		var conditionReason string
		var conditionMessage string
		failed := false
		for _, condition := range tr.Status.Conditions {
			if strings.EqualFold(string(condition.Type), "Succeeded") && condition.Status == "False" {
				failed = true
				conditionReason = strings.TrimSpace(condition.Reason)
				conditionMessage = strings.TrimSpace(condition.Message)
				break
			}
		}

		stepName := ""
		stepReason := ""
		stepMessage := ""
		for _, step := range tr.Status.Steps {
			if step.Terminated != nil && step.Terminated.ExitCode != 0 {
				failed = true
				stepName = strings.TrimSpace(step.Name)
				if stepName == "" {
					stepName = strings.TrimSpace(step.Container)
				}
				stepReason = strings.TrimSpace(step.Terminated.Reason)
				stepMessage = strings.TrimSpace(step.Terminated.Message)
				break
			}
		}

		if !failed {
			continue
		}

		message := conditionMessage
		if message == "" {
			message = stepMessage
		}
		if message == "" {
			message = "TaskRun failed"
		}
		reason := conditionReason
		if reason == "" {
			reason = stepReason
		}

		return &tektonFailureSummary{
			PipelineTask: strings.TrimSpace(tr.Labels["tekton.dev/pipelineTask"]),
			TaskRun:      tr.Name,
			Step:         stepName,
			Pod:          strings.TrimSpace(tr.Status.PodName),
			Stage:        inferExecutionStage(tr.Name, stepName),
			Reason:       reason,
			Message:      message,
		}
	}

	return nil
}

func (e *MethodTektonExecutor) finalizeExecutionFromPipelineRun(
	ctx context.Context,
	executionID uuid.UUID,
	pr *tektonv1.PipelineRun,
	failureMessage string,
	tektonClient tektonclient.Interface,
) {
	if len(pr.Status.Conditions) == 0 {
		e.finalizeExecution(ctx, executionID, ExecutionFailed, "PipelineRun completed with unknown status", nil)
		return
	}

	status := pr.Status.Conditions[0]
	var execStatus ExecutionStatus
	var message string
	var artifacts []Artifact

	switch status.Status {
	case "True":
		execStatus = ExecutionSuccess
		message = "Pipeline completed successfully"
		artifacts = e.collectArtifacts(ctx, pr, tektonClient)
	case "False":
		execStatus = ExecutionFailed
		message = strings.TrimSpace(failureMessage)
		if message == "" {
			message = status.Message
			if status.Reason != "" {
				message += fmt.Sprintf(" (%s)", status.Reason)
			}
		}
	default:
		execStatus = ExecutionFailed
		message = "Pipeline status unknown"
	}

	e.finalizeExecution(ctx, executionID, execStatus, message, artifacts)
}

func (e *MethodTektonExecutor) finalizeExecution(ctx context.Context, executionID uuid.UUID, status ExecutionStatus, message string, artifacts []Artifact) {
	logLevel := LogInfo
	if status == ExecutionFailed {
		logLevel = LogError
	}

	var metadata []byte
	if len(artifacts) > 0 {
		metadata, _ = json.Marshal(artifacts)
	}

	e.service.AddLog(ctx, executionID, logLevel, message, metadata)

	switch status {
	case ExecutionSuccess:
		if err := e.service.CompleteExecution(ctx, executionID, true, "", metadata); err != nil {
			e.logger.Error("Failed to complete execution as success", zap.Error(err))
		}
	case ExecutionFailed:
		if err := e.service.CompleteExecution(ctx, executionID, false, message, metadata); err != nil {
			e.logger.Error("Failed to complete execution as failed", zap.Error(err))
		}
	default:
		if err := e.service.UpdateExecutionStatus(ctx, executionID, status); err != nil {
			e.logger.Error("Failed to update execution status", zap.Error(err))
		}
	}

	e.syncBuildStatusFromExecution(context.Background(), executionID, status, message)

	e.logger.Info("Execution finalized",
		zap.String("executionID", executionID.String()),
		zap.String("status", string(status)),
		zap.String("message", message))
}

func (e *MethodTektonExecutor) syncBuildStatusFromExecution(ctx context.Context, executionID uuid.UUID, status ExecutionStatus, message string) {
	if e.buildRepo == nil {
		return
	}
	execution, err := e.service.GetExecution(ctx, executionID)
	if err != nil || execution == nil {
		e.logger.Warn("Failed to load execution for build status sync", zap.String("execution_id", executionID.String()), zap.Error(err))
		return
	}
	build, err := e.buildRepo.FindByID(ctx, execution.BuildID)
	if err != nil || build == nil {
		e.logger.Warn("Failed to load build for status sync",
			zap.String("execution_id", executionID.String()),
			zap.String("build_id", execution.BuildID.String()),
			zap.Error(err))
		return
	}
	if build.IsTerminal() {
		return
	}

	switch status {
	case ExecutionSuccess:
		if err := build.Complete(BuildResult{Logs: []string{message}}); err != nil {
			e.logger.Warn("Failed to mark build completed from execution",
				zap.String("execution_id", executionID.String()),
				zap.String("build_id", build.ID().String()),
				zap.Error(err))
			return
		}
	case ExecutionFailed:
		if err := build.Fail(message); err != nil {
			e.logger.Warn("Failed to mark build failed from execution",
				zap.String("execution_id", executionID.String()),
				zap.String("build_id", build.ID().String()),
				zap.Error(err))
			return
		}
	case ExecutionCancelled:
		if err := build.Cancel(); err != nil {
			e.logger.Warn("Failed to mark build cancelled from execution",
				zap.String("execution_id", executionID.String()),
				zap.String("build_id", build.ID().String()),
				zap.Error(err))
			return
		}
	default:
		return
	}

	if err := e.buildRepo.Update(ctx, build); err != nil {
		e.logger.Warn("Failed to persist build status sync from execution",
			zap.String("execution_id", executionID.String()),
			zap.String("build_id", build.ID().String()),
			zap.Error(err))
		return
	}

	if emitter, ok := e.service.(BuildStatusUpdateEmitter); ok {
		emitter.EmitBuildStatusUpdate(ctx, build.ID(), string(build.Status()), message, map[string]interface{}{
			"execution_id": executionID.String(),
			"source":       "tekton_sync",
		})
	}
}

func (e *MethodTektonExecutor) collectArtifacts(
	ctx context.Context,
	pr *tektonv1.PipelineRun,
	tektonClient tektonclient.Interface,
) []Artifact {
	if pr == nil {
		return nil
	}
	artifacts := make([]Artifact, 0, 16)
	seen := make(map[string]struct{}, 16)

	appendArtifact := func(artifact Artifact) {
		name := strings.TrimSpace(artifact.Name)
		value := strings.TrimSpace(artifact.Value)
		if name == "" || value == "" {
			return
		}
		artifact.Name = name
		artifact.Value = value
		key := strings.Join([]string{artifact.Type, artifact.Name, artifact.Value}, "|")
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		artifacts = append(artifacts, artifact)
	}

	for _, result := range pr.Status.Results {
		appendArtifact(Artifact{
			Name:  strings.TrimSpace(result.Name),
			Type:  "pipeline-result",
			Value: tektonResultValueString(result.Value),
		})
	}

	if tektonClient != nil && strings.TrimSpace(pr.Namespace) != "" && strings.TrimSpace(pr.Name) != "" {
		taskRuns, err := tektonClient.TektonV1().TaskRuns(pr.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", pr.Name),
		})
		if err != nil {
			e.logger.Warn("Failed to list TaskRuns for artifact collection",
				zap.String("namespace", pr.Namespace),
				zap.String("pipeline_run", pr.Name),
				zap.Error(err))
		} else {
			for _, tr := range taskRuns.Items {
				taskPrefix := strings.TrimSpace(tr.Labels["tekton.dev/pipelineTask"])
				if taskPrefix == "" {
					taskPrefix = strings.TrimSpace(tr.Name)
				}
				for _, result := range tr.Status.Results {
					name := strings.TrimSpace(result.Name)
					if taskPrefix != "" {
						name = fmt.Sprintf("%s.%s", taskPrefix, name)
					}
					appendArtifact(Artifact{
						Name:  name,
						Type:  "taskrun-result",
						Value: tektonResultValueString(result.Value),
					})
				}
			}
		}
	}

	return artifacts
}

func tektonResultValueString(value tektonv1.ResultValue) string {
	raw := strings.TrimSpace(value.StringVal)
	if raw != "" {
		return raw
	}
	if len(value.ArrayVal) > 0 {
		encoded, err := json.Marshal(value.ArrayVal)
		if err == nil {
			return strings.TrimSpace(string(encoded))
		}
	}
	if len(value.ObjectVal) > 0 {
		encoded, err := json.Marshal(value.ObjectVal)
		if err == nil {
			return strings.TrimSpace(string(encoded))
		}
	}
	return ""
}
