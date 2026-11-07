import { api } from "./bridge";

// Resolve project saves/discards first. Cancelling that dialog must leave the
// capture running; the native close retry receives both approvals together.
export function requestSessionQuit(
  closeProjects: (done: () => void) => void,
  onError: (message: string) => void,
) {
  closeProjects(() => {
    void api.StopCommand()
      .then(() => api.ConfirmSessionClose())
      .catch((error) => onError(String(error)));
  });
}
