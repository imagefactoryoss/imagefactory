import React from "react";

import type {
    LDAPConfigFormData,
    SMTPConfigFormData,
} from "./systemConfigurationShared";

export const LDAPConfigurationPanel: React.FC<{
    ldapConfig: LDAPConfigFormData;
    setLdapConfig: React.Dispatch<React.SetStateAction<LDAPConfigFormData>>;
    newDomain: string;
    setNewDomain: React.Dispatch<React.SetStateAction<string>>;
    canManageAdmin: boolean;
    addAllowedDomain: () => void;
    removeAllowedDomain: (domain: string) => void;
    saveLdapConfig: () => void;
}> = ({
    ldapConfig,
    setLdapConfig,
    newDomain,
    setNewDomain,
    canManageAdmin,
    addAllowedDomain,
    removeAllowedDomain,
    saveLdapConfig,
}) => (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                LDAP Authentication Settings
            </h2>
            <div className="mb-6 flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 px-4 py-3">
                <div>
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                        LDAP Authentication
                    </h3>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                        Disable to turn off LDAP login and directory lookups.
                    </p>
                </div>
                <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                    <input
                        type="checkbox"
                        checked={ldapConfig.enabled}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                enabled: e.target.checked,
                            }))
                        }
                        className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                    />
                    {ldapConfig.enabled ? "Enabled" : "Disabled"}
                </label>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        LDAP Host
                    </label>
                    <input
                        type="text"
                        value={ldapConfig.host}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({ ...prev, host: e.target.value }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        LDAP Port
                    </label>
                    <input
                        type="number"
                        value={ldapConfig.port}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                port: parseInt(e.target.value),
                            }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Base DN
                    </label>
                    <input
                        type="text"
                        value={ldapConfig.base_dn}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                base_dn: e.target.value,
                            }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Bind DN
                    </label>
                    <input
                        type="text"
                        value={ldapConfig.bind_dn}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                bind_dn: e.target.value,
                            }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        User Search Base
                    </label>
                    <input
                        type="text"
                        value={ldapConfig.user_search_base}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                user_search_base: e.target.value,
                            }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Group Search Base
                    </label>
                    <input
                        type="text"
                        value={ldapConfig.group_search_base}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                group_search_base: e.target.value,
                            }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Bind Password
                    </label>
                    <input
                        type="password"
                        value={ldapConfig.bind_password}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                bind_password: e.target.value,
                            }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        User Filter
                    </label>
                    <input
                        type="text"
                        value={ldapConfig.user_filter}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                user_filter: e.target.value,
                            }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Group Filter
                    </label>
                    <input
                        type="text"
                        value={ldapConfig.group_filter}
                        onChange={(e) =>
                            setLdapConfig((prev) => ({
                                ...prev,
                                group_filter: e.target.value,
                            }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                </div>
                <div className="flex items-center space-x-4">
                    <label className="flex items-center">
                        <input
                            type="checkbox"
                            checked={ldapConfig.start_tls}
                            onChange={(e) =>
                                setLdapConfig((prev) => ({
                                    ...prev,
                                    start_tls: e.target.checked,
                                }))
                            }
                            className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                        />
                        <span className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                            Start TLS
                        </span>
                    </label>
                    <label className="flex items-center">
                        <input
                            type="checkbox"
                            checked={ldapConfig.ssl}
                            onChange={(e) =>
                                setLdapConfig((prev) => ({
                                    ...prev,
                                    ssl: e.target.checked,
                                }))
                            }
                            className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                        />
                        <span className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                            SSL
                        </span>
                    </label>
                </div>
            </div>

            <div className="mt-6">
                <h3 className="text-md font-medium text-slate-900 dark:text-white mb-4">
                    Allowed Email Domains
                </h3>
                <div className="space-y-4">
                    <div className="flex gap-2">
                        <input
                            type="text"
                            value={newDomain}
                            onChange={(e) => setNewDomain(e.target.value)}
                            placeholder="@example.com"
                            className="flex-1 px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                            onKeyPress={(e) =>
                                e.key === "Enter" && canManageAdmin && addAllowedDomain()
                            }
                        />
                        {canManageAdmin && (
                            <button
                                onClick={addAllowedDomain}
                                className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-md font-medium transition-colors"
                            >
                                Add Domain
                            </button>
                        )}
                    </div>
                    {ldapConfig.allowed_domains.length > 0 && (
                        <div className="space-y-2">
                            <p className="text-sm text-slate-600 dark:text-slate-400">
                                Current allowed domains:
                            </p>
                            <div className="flex flex-wrap gap-2">
                                {ldapConfig.allowed_domains.map((domain, index) => (
                                    <div
                                        key={index}
                                        className="flex items-center bg-slate-100 dark:bg-slate-700 rounded-md px-3 py-1"
                                    >
                                        <span className="text-sm text-slate-900 dark:text-white">
                                            {domain}
                                        </span>
                                        {canManageAdmin && (
                                            <button
                                                onClick={() => removeAllowedDomain(domain)}
                                                className="ml-2 text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                                            >
                                                ×
                                            </button>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </div>

            <div className="mt-6">
                {canManageAdmin && (
                    <button
                        onClick={saveLdapConfig}
                        className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
                    >
                        Save LDAP Configuration
                    </button>
                )}
            </div>
        </div>
    );

export const SMTPConfigurationPanel: React.FC<{
    smtpConfig: SMTPConfigFormData;
    setSmtpConfig: React.Dispatch<React.SetStateAction<SMTPConfigFormData>>;
    canManageAdmin: boolean;
    saveSmtpConfig: () => void;
}> = ({ smtpConfig, setSmtpConfig, canManageAdmin, saveSmtpConfig }) => (
    <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            SMTP Email Settings
        </h2>
        <div className="mb-6 flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 px-4 py-3">
            <div>
                <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                    SMTP Delivery
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                    Disable to stop sending email notifications.
                </p>
            </div>
            <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                <input
                    type="checkbox"
                    checked={smtpConfig.enabled}
                    onChange={(e) =>
                        setSmtpConfig((prev) => ({
                            ...prev,
                            enabled: e.target.checked,
                        }))
                    }
                    className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                />
                {smtpConfig.enabled ? "Enabled" : "Disabled"}
            </label>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    SMTP Host
                </label>
                <input
                    type="text"
                    value={smtpConfig.host}
                    onChange={(e) =>
                        setSmtpConfig((prev) => ({ ...prev, host: e.target.value }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    SMTP Port
                </label>
                <input
                    type="number"
                    value={smtpConfig.port}
                    onChange={(e) =>
                        setSmtpConfig((prev) => ({
                            ...prev,
                            port: parseInt(e.target.value),
                        }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Username
                </label>
                <input
                    type="text"
                    value={smtpConfig.username}
                    onChange={(e) =>
                        setSmtpConfig((prev) => ({
                            ...prev,
                            username: e.target.value,
                        }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Password
                </label>
                <input
                    type="password"
                    value={smtpConfig.password}
                    onChange={(e) =>
                        setSmtpConfig((prev) => ({
                            ...prev,
                            password: e.target.value,
                        }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    From Email
                </label>
                <input
                    type="email"
                    value={smtpConfig.from}
                    onChange={(e) =>
                        setSmtpConfig((prev) => ({ ...prev, from: e.target.value }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
            </div>
            <div className="flex items-center space-x-4">
                <label className="flex items-center">
                    <input
                        type="checkbox"
                        checked={smtpConfig.start_tls}
                        onChange={(e) =>
                            setSmtpConfig((prev) => ({
                                ...prev,
                                start_tls: e.target.checked,
                            }))
                        }
                        className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                    />
                    <span className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                        Start TLS
                    </span>
                </label>
                <label className="flex items-center">
                    <input
                        type="checkbox"
                        checked={smtpConfig.ssl}
                        onChange={(e) =>
                            setSmtpConfig((prev) => ({
                                ...prev,
                                ssl: e.target.checked,
                            }))
                        }
                        className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                    />
                    <span className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                        SSL
                    </span>
                </label>
            </div>
        </div>
        <div className="mt-6">
            {canManageAdmin && (
                <button
                    onClick={saveSmtpConfig}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
                >
                    Save SMTP Configuration
                </button>
            )}
        </div>
    </div>
);