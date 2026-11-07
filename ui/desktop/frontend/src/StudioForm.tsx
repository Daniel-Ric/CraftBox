import type { ComponentProps } from "react";
import { DialogValidation, useDialogValidation } from "./form-validation";
function Form({ onSubmit, ...props }: ComponentProps<"form">) {
  const validation = useDialogValidation();
  return (
    <form
      {...props}
      onSubmit={(e) => {
        if (validation?.invalid || !e.currentTarget.checkValidity()) {
          e.preventDefault();
          e.currentTarget.reportValidity();
          return;
        }
        onSubmit?.(e);
      }}
    />
  );
}
export default function StudioForm(props: ComponentProps<"form">) {
  return (
    <DialogValidation>
      <Form {...props} />
    </DialogValidation>
  );
}
