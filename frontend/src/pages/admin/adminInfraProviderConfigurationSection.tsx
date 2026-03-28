import { Eye, EyeOff } from "lucide-react";
import React from "react";

type InfraProviderConfigurationSectionProps = {
  configEntries: Array<[string, unknown]>;
  visibleAuthConfig?: Record<string, unknown>;
  visibleAuthLabel: string;
  isManagedBootstrapProvider: boolean;
  visibleSensitiveFields: Record<string, boolean>;
  isSensitiveKey: (key: string) => boolean;
  toggleSensitiveField: (fieldPath: string) => void;
};

const formatLabel = (value: string) =>
  value.replace(/([A-Z])/g, " $1").replace(/^./, (first) => first.toUpperCase());

const maskValue = "••••••••••••••••••••••••••••••••";

const InfraProviderConfigurationSection: React.FC<
  InfraProviderConfigurationSectionProps
> = ({
  configEntries,
  visibleAuthConfig,
  visibleAuthLabel,
  isManagedBootstrapProvider,
  visibleSensitiveFields,
  isSensitiveKey,
  toggleSensitiveField,
}) => {
  const hasVisibleAuthConfig =
    !!visibleAuthConfig && Object.keys(visibleAuthConfig).length > 0;

  return (
    <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
      <div className="px-4 py-5 sm:p-6">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
          Configuration
        </h3>
        {configEntries.length > 0 || hasVisibleAuthConfig ? (
          <dl className="space-y-4">
            {hasVisibleAuthConfig ? (
              <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 p-3">
                <dt className="text-sm font-medium text-gray-700 dark:text-gray-200 mb-2">
                  {visibleAuthLabel}
                </dt>
                <div className="space-y-2">
                  {Object.entries(visibleAuthConfig || {}).map(([authKey, authValue]) => {
                    const fieldPath = `${
                      isManagedBootstrapProvider ? "bootstrap_auth" : "runtime_auth"
                    }.${authKey}`;
                    const isSensitive = isSensitiveKey(authKey);
                    const isVisible = visibleSensitiveFields[fieldPath];
                    return (
                      <div key={fieldPath}>
                        <div className="text-sm font-medium text-gray-500 dark:text-gray-400 capitalize flex items-center justify-between">
                          <span>{formatLabel(authKey)}</span>
                          {isSensitive ? (
                            <button
                              onClick={() => toggleSensitiveField(fieldPath)}
                              className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 p-1 rounded"
                              title={isVisible ? "Hide sensitive data" : "Show sensitive data"}
                            >
                              {isVisible ? (
                                <EyeOff className="h-4 w-4" />
                              ) : (
                                <Eye className="h-4 w-4" />
                              )}
                            </button>
                          ) : null}
                        </div>
                        <div className="mt-1 text-sm text-gray-900 dark:text-white font-mono bg-white dark:bg-gray-800 p-2 rounded break-all">
                          {isSensitive && !isVisible
                            ? maskValue
                            : typeof authValue === "object"
                              ? JSON.stringify(authValue, null, 2)
                              : String(authValue)}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            ) : null}
            {configEntries.map(([key, value]) => (
              <div key={key}>
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400 capitalize flex items-center justify-between">
                  <span>{formatLabel(key)}</span>
                  {isSensitiveKey(key) ? (
                    <button
                      onClick={() => toggleSensitiveField(key)}
                      className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 p-1 rounded"
                      title={
                        visibleSensitiveFields[key]
                          ? "Hide sensitive data"
                          : "Show sensitive data"
                      }
                    >
                      {visibleSensitiveFields[key] ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </button>
                  ) : null}
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white font-mono bg-gray-50 dark:bg-gray-700 p-2 rounded break-all">
                  {isSensitiveKey(key) && !visibleSensitiveFields[key]
                    ? maskValue
                    : typeof value === "object"
                      ? JSON.stringify(value, null, 2)
                      : String(value)}
                </dd>
              </div>
            ))}
          </dl>
        ) : (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            No configuration details available
          </p>
        )}
      </div>
    </div>
  );
};

export default InfraProviderConfigurationSection;
