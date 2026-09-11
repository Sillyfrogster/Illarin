export function replacementSubjectLabel(subject: string): string {
  if (subject === "images" || subject === "pictures") return "Pictures";
  if (subject === "opaque_data" || subject === "preserved_data") {
    return "Preserved data";
  }
  const words = subject.replaceAll("_", " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}
