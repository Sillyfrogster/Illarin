"use client";

import { useState } from "react";
import { Switch } from "@/components/ui/switch";
import { useAuth } from "@/lib/auth";

/** ArtworkSwitch removes every piece of art on the site, and follows a signed-in reader to other browsers. */
export function ArtworkSwitch({ hint }: { hint?: string }) {
  const { artwork, setArtwork } = useAuth();
  const [failed, setFailed] = useState(false);

  return (
    <div>
      <Switch
        checked={artwork}
        hint={hint}
        label="Artwork"
        onCheckedChange={(on) => {
          setFailed(false);
          setArtwork(on).catch(() => setFailed(true));
        }}
      />
      {failed ? (
        <p role="alert" className="text-meta text-stop">
          Your choice applies here but could not be saved to your account.
        </p>
      ) : null}
    </div>
  );
}
