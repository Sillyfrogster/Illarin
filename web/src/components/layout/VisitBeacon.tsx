"use client";

import { useEffect } from "react";
import { api } from "@/lib/api/client";

/** Tells the API a reader opened the site, which counts once per reader per day */
export function VisitBeacon() {
  useEffect(() => {
    api("POST", "/v1/visits").catch(() => {});
  }, []);
  return null;
}
