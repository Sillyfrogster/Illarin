"use client";

import { createContext, type ReactNode, useContext } from "react";
import type { ExtensionDependency } from "@/lib/api/query";

const NO_DEPENDENCIES: ExtensionDependency[] = [];

const ExtensionDependenciesContext =
  createContext<ExtensionDependency[]>(NO_DEPENDENCIES);

export function ExtensionDependenciesProvider({
  dependencies,
  children,
}: {
  dependencies: ExtensionDependency[];
  children: ReactNode;
}) {
  return (
    <ExtensionDependenciesContext.Provider value={dependencies}>
      {children}
    </ExtensionDependenciesContext.Provider>
  );
}

export function useExtensionDependencies(): ExtensionDependency[] {
  return useContext(ExtensionDependenciesContext);
}

/** Pairs each dependency the archive names with the listed extensions the page was read with. */
export function dependencyLinks(
  texts: { text: string }[],
  dependencies: ExtensionDependency[],
): ExtensionDependency[] {
  const matched = new Map(
    dependencies.map((dependency) => [dependency.name, dependency.works]),
  );
  return texts.map(({ text }) => ({
    name: text,
    works: matched.get(text) ?? [],
  }));
}
