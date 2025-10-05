import { formatCurrency } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { StatusHistory } from "./StatusHistory";
import { Comments } from "@/components/claims/Comments";
import React,
{
	useState,
	useEffect
}
from "react";

interface Field < TData > {
	key: keyof TData;
	label: string;
	type ? : 'currency';
	options ? : string[];
}

interface DetailsDrawerProps < TData > {
	data: TData;
	fields: {
		main: Field < TData > [];
		status: Field < TData > [];
		comments: Field < TData > [];
		// ADDED: A new field for full-width content
		fullWidth ? : Field < TData > [];
	};
	onSave: (updatedData: TData) => void;
	onCancel: () => void;
	id ? : number;
	type ? : 'item';
}

export function DetailsDrawer < TData extends object > ({
	data,
	fields,
	onSave,
	onCancel,
	id,
	type
}: DetailsDrawerProps < TData > ) {
	const [editableData, setEditableData] = useState(data);
	const [hasChanges, setHasChanges] = useState(false);

	useEffect(() => {
		setEditableData(data);
	}, [data]);

	useEffect(() => {
		setHasChanges(JSON.stringify(data) !== JSON.stringify(editableData));
	}, [data, editableData]);

	const handleInputChange = (key: keyof TData, value: any) => {
		let processedValue = value;
		if (key === 'poc') {
			if (value === '') {
				processedValue = null;
			} else if (typeof value === 'string') {
				processedValue = Number(value);
			}
		}
		setEditableData(prev => ({
			...prev,
			[key]: processedValue
		}));
	};

	const handleCancel = () => {
		setEditableData(data);
		onCancel();
	};

	const handleSave = () => {
		onSave(editableData);
	};

	const renderValue = (value: any, type ? : 'currency') => {
		if (value === null || value === undefined) {
			return "";
		}
		if (type === 'currency' && typeof value === 'number') {
			return formatCurrency(value);
		}
		return String(value);
	};

	return ( <
		div className = "p-4 flex flex-col h-full" >
		<
		div className = "grid grid-cols-2 gap-4" >
		<
		div >
		<
		h3 className = "text-lg font-semibold mb-2" > Details < /h3> <
		div className = "grid grid-cols-2 gap-x-4" > {
			fields.main.map((field) => ( <
				React.Fragment key = {
					String(field.key)
				} >
				<
				strong className = "text-right" > {
					field.label
				}: < /strong> <
				span > {
					renderValue(data[field.key], field.type)
				} < /span> <
				/React.Fragment>
			))
		} <
		/div> <
		/div> <
		div >
		<
		h3 className = "text-lg font-semibold mb-2" > Status < /h3> {
			fields.status.map((field) => ( <
				div key = {
					String(field.key)
				}
				className = "mb-2" >
				<
				label className = "font-semibold" > {
					field.label
				}: < /label> {
					field.options ? ( <
						Select onValueChange = {
							(value) => handleInputChange(field.key, value)
						}
						value = {
							renderValue(editableData[field.key])
						} >
						<
						SelectTrigger >
						<
						SelectValue placeholder = {
							`Select a ${field.label}`
						}
						/> <
						/SelectTrigger> <
						SelectContent > {
							field.options.map(option => ( <
								SelectItem key = {
									option
								}
								value = {
									option
								} > {
									option
								} < /SelectItem>
							))
						} <
						/SelectContent> <
						/Select>
					) : ( <
						Input value = {
							renderValue(editableData[field.key])
						}
						onChange = {
							(e) => handleInputChange(field.key, e.target.value)
						}
						/>
					)
				} <
				/div>
			))
		} {
			hasChanges && ( <
				div className = "flex justify-end space-x-2 mt-4" >
				<
				Button variant = "outline"
				onClick = {
					handleCancel
				} > Cancel < /Button> <
				Button onClick = {
					handleSave
				} > Save < /Button> <
				/div>
			)
		} <
		/div> <
		/div>

		{ /* ADDED: Section for full-width fields */ } {
			fields.fullWidth && fields.fullWidth.length > 0 && ( <
				div className = "mt-4 pt-4 border-t" > {
					fields.fullWidth.map((field) => ( <
						div key = {
							String(field.key)
						}
						className = "mb-4" >
						<
						h3 className = "text-lg font-semibold mb-2" > {
							field.label
						} < /h3> <
						p className = "text-sm text-muted-foreground whitespace-pre-wrap" > {
							renderValue(data[field.key])
						} < /p> <
						/div>
					))
				} <
				/div>
			)
		}

		<
		div className = "mt-4 pt-4 border-t flex-grow" >
		<
		Accordion type = "single"
		collapsible defaultValue = "status-history" >
		<
		AccordionItem value = "comments" >
		<
		AccordionTrigger > Comments < /AccordionTrigger> <
		AccordionContent > {
			id && < Comments itemId = {
				id
			}
			/>} <
			/AccordionContent> <
			/AccordionItem> <
			AccordionItem value = "status-history" >
			<
			AccordionTrigger > Status History < /AccordionTrigger> <
			AccordionContent > {
				id && type && < StatusHistory id = {
					id
				}
				type = "items" / >
			} <
			/AccordionContent> <
			/AccordionItem> <
			/Accordion> <
			/div> <
			/div>
	);
}
