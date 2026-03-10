import { BUILD_METHODS } from '@/lib/buildMethods';
import { BuildMethod, BuildMethodInfo } from '@/types/buildConfig';
import React, { useState } from 'react';

interface BuildMethodSelectorProps {
    onSelect: (method: BuildMethod) => void;
    selected?: BuildMethod;
    showRequirements?: boolean;
}

const BuildMethodSelector: React.FC<BuildMethodSelectorProps> = ({
    onSelect,
    selected,
    showRequirements = true,
}) => {
    const [expandedMethod, setExpandedMethod] = useState<BuildMethod | null>(selected || null);

    const handleMethodClick = (method: BuildMethod) => {
        setExpandedMethod(expandedMethod === method ? null : method);
    };

    const handleSelectMethod = (method: BuildMethod) => {
        onSelect(method);
        setExpandedMethod(method);
    };

    return (
        <div className="build-method-selector">
            <div className="mb-6">
                <h2 className="text-2xl font-bold mb-2">Select Build Method</h2>
                <p className="text-gray-600">Choose the build method that best fits your needs</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {Object.entries(BUILD_METHODS).map(([key, method]) => (
                    <MethodCard
                        key={key}
                        method={method}
                        isSelected={selected === key}
                        isExpanded={expandedMethod === key}
                        onSelect={() => handleSelectMethod(key as BuildMethod)}
                        onExpand={() => handleMethodClick(key as BuildMethod)}
                        showRequirements={showRequirements}
                    />
                ))}
            </div>
        </div>
    );
};

interface MethodCardProps {
    method: BuildMethodInfo;
    isSelected: boolean;
    isExpanded: boolean;
    onSelect: () => void;
    onExpand: () => void;
    showRequirements: boolean;
}

const MethodCard: React.FC<MethodCardProps> = ({
    method,
    isSelected,
    isExpanded,
    onSelect,
    onExpand,
    showRequirements,
}) => {
    return (
        <div
            className={`
        border rounded-lg p-4 transition-all cursor-pointer
        ${isSelected ? 'border-blue-500 bg-blue-50 shadow-lg' : 'border-gray-200 bg-white hover:shadow-md'}
      `}
        >
            {/* Header */}
            <div className="flex items-start justify-between mb-3" onClick={onExpand}>
                <div className="flex items-center gap-3 flex-1">
                    <span className="text-3xl">{method.icon}</span>
                    <div>
                        <h3 className="font-bold text-lg">{method.name}</h3>
                        <p className="text-sm text-gray-600">{method.description}</p>
                    </div>
                </div>
                <button
                    onClick={(e) => {
                        e.stopPropagation();
                        onSelect();
                    }}
                    className={`
            px-4 py-2 rounded text-sm font-medium transition-colors
            ${isSelected
                            ? 'bg-blue-500 text-white'
                            : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                        }
          `}
                >
                    {isSelected ? '✓ Selected' : 'Select'}
                </button>
            </div>

            {/* Best For */}
            <div className="mb-3 text-sm">
                <span className="font-semibold text-gray-700">Best for:</span>
                <p className="text-gray-600 mt-1">{method.bestFor}</p>
            </div>

            {/* Expandable Details */}
            {isExpanded && (
                <div className="mt-4 pt-4 border-t border-gray-200 space-y-4">
                    {/* Advantages */}
                    <div>
                        <h4 className="font-semibold text-gray-700 mb-2">Advantages</h4>
                        <ul className="space-y-1">
                            {method.advantages.map((advantage, idx) => (
                                <li key={idx} className="text-sm text-gray-600 flex items-start">
                                    <span className="mr-2">✓</span>
                                    <span>{advantage}</span>
                                </li>
                            ))}
                        </ul>
                    </div>

                    {/* Requirements */}
                    {showRequirements && (
                        <div>
                            <h4 className="font-semibold text-gray-700 mb-2">Requirements</h4>
                            <ul className="space-y-1">
                                {method.requirements.map((requirement, idx) => (
                                    <li key={idx} className="text-sm text-gray-600 flex items-start">
                                        <span className="mr-2">•</span>
                                        <span>{requirement}</span>
                                    </li>
                                ))}
                            </ul>
                        </div>
                    )}

                    {/* Documentation Link */}
                    <div>
                        <a
                            href={method.documentationUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-sm text-blue-600 hover:text-blue-800 underline"
                        >
                            View Documentation →
                        </a>
                    </div>
                </div>
            )}
        </div>
    );
};

export default BuildMethodSelector;
