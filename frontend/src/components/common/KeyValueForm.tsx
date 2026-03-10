import React from 'react';

interface KeyValuePair {
    key: string;
    value: string;
}

interface KeyValueFormProps {
    title: string;
    items: KeyValuePair[];
    onItemsChange: (items: KeyValuePair[]) => void;
    keyPlaceholder?: string;
    valuePlaceholder?: string;
    valueType?: 'text' | 'password' | 'url';
    onlyKeys?: boolean;
}

export const KeyValueForm: React.FC<KeyValueFormProps> = ({
    title,
    items,
    onItemsChange,
    keyPlaceholder = 'Key',
    valuePlaceholder = 'Value',
    valueType = 'text',
    onlyKeys = false,
}) => {
    const handleAddItem = () => {
        onItemsChange([...items, { key: '', value: '' }]);
    };

    const handleRemoveItem = (index: number) => {
        onItemsChange(items.filter((_, i) => i !== index));
    };

    const handleItemChange = (index: number, field: 'key' | 'value', value: string) => {
        const updated = [...items];
        updated[index] = { ...updated[index], [field]: value };
        onItemsChange(updated);
    };

    return (
        <div className="space-y-3">
            <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300">{title}</label>

            <div className="space-y-2">
                {items.map((item, idx) => (
                    <div key={idx} className="flex gap-2 items-end">
                        <div className="flex-1">
                            <input
                                type="text"
                                placeholder={keyPlaceholder}
                                value={item.key}
                                onChange={(e) => handleItemChange(idx, 'key', e.target.value)}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                            />
                        </div>

                        {!onlyKeys && (
                            <div className="flex-1">
                                <input
                                    type={valueType}
                                    placeholder={valuePlaceholder}
                                    value={item.value}
                                    onChange={(e) => handleItemChange(idx, 'value', e.target.value)}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400"
                                />
                            </div>
                        )}

                        <button
                            onClick={() => handleRemoveItem(idx)}
                            className="px-3 py-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-md text-sm font-medium transition-colors"
                            type="button"
                        >
                            Remove
                        </button>
                    </div>
                ))}
            </div>

            <button
                onClick={handleAddItem}
                type="button"
                className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 font-medium"
            >
                + Add {onlyKeys ? 'Item' : 'Pair'}
            </button>
        </div>
    );
};

export default KeyValueForm;
