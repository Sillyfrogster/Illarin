/** versionTag shows a creator's version label with one leading "v", whether or not they typed it. */
export const versionTag = (label: string): string =>
  /^v/i.test(label) ? label : `v${label}`;
