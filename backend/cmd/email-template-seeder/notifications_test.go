package main

import "testing"

func TestGetNotificationTemplates_ContainsImageImportWorkflowTemplates(t *testing.T) {
	templates := getNotificationTemplates()
	byType := make(map[string]EmailTemplate, len(templates))
	for _, tmpl := range templates {
		byType[tmpl.TemplateType] = tmpl
	}

	requiredTypes := []string{
		"external_image_import_approval_requested",
		"external_image_import_approved",
		"external_image_import_rejected",
		"external_image_import_dispatch_failed",
		"external_image_import_completed",
		"external_image_import_quarantined",
		"external_image_import_failed",
		"epr_registration_requested",
		"epr_registration_approved",
		"epr_registration_rejected",
		"epr_registration_suspended",
		"epr_registration_reactivated",
		"epr_registration_revalidated",
		"epr_registration_expiring",
		"epr_registration_expired",
	}

	for _, templateType := range requiredTypes {
		tmpl, ok := byType[templateType]
		if !ok {
			t.Fatalf("missing required workflow notification template: %s", templateType)
		}
		if tmpl.Category != CategoryNotifications {
			t.Fatalf("template %s has unexpected category %s", templateType, tmpl.Category)
		}
		if tmpl.Subject == "" || tmpl.TextTemplate == "" || tmpl.HTMLTemplate == "" {
			t.Fatalf("template %s is missing required content fields", templateType)
		}
	}
}
