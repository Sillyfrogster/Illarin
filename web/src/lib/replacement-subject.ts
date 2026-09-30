export function replacementSubjectLabel(subject: string): string {
  if (subject === "images" || subject === "pictures") return "Images";
  if (subject === "opaque_data" || subject === "preserved_data") {
    return "File extras";
  }
  const words = subject.replaceAll("_", " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}
