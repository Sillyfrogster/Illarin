import { BROWSER_MUTATION_HEADER } from "@/lib/api/browser-mutation";
import type { UploadOperation } from "@/lib/api/query";

export type Sent =
  | { operation: UploadOperation; error?: undefined }
  | { operation?: undefined; error: string };

const UNREACHABLE =
  "Illarin could not be reached. Check your connection and try again.";

/** sendFile posts a file as a new work, reporting the share of bytes sent, since fetch cannot report upload progress. */
export function sendFile(
  file: File,
  onSent: (fraction: number) => void,
): Promise<Sent> {
  const body = new FormData();
  body.append("metadata", JSON.stringify({ confirmed: true }));
  body.append("file", file, file.name);

  return new Promise((resolve) => {
    const request = new XMLHttpRequest();
    request.open("POST", "/api/v1/works");
    request.setRequestHeader(BROWSER_MUTATION_HEADER, "1");
    request.upload.onprogress = (event) => {
      if (event.lengthComputable) onSent(event.loaded / event.total);
    };
    request.onerror = () => resolve({ error: UNREACHABLE });
    request.onload = () => {
      let answer: unknown;
      try {
        answer = JSON.parse(request.responseText);
      } catch {
        answer = null;
      }
      if (request.status >= 200 && request.status < 300 && answer) {
        resolve({ operation: answer as UploadOperation });
        return;
      }
      const said = (answer as { error?: string } | null)?.error;
      resolve({ error: said ?? "Illarin could not accept this file." });
    };
    request.send(body);
  });
}
