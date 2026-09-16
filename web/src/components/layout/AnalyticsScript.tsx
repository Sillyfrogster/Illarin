/** The deploy creates the Umami website under this id */
const ANALYTICS_WEBSITE_ID = "52bac8b6-2dd7-40c0-91a3-5b04e7298345";

type AnalyticsScriptProps = {
  production?: boolean;
};

/** Loads the Umami tracker without query strings or fragments */
export function AnalyticsScript({
  production = process.env.NODE_ENV === "production",
}: AnalyticsScriptProps) {
  if (!production) {
    return null;
  }
  return (
    <script
      defer
      src="/stats/script.js"
      data-website-id={ANALYTICS_WEBSITE_ID}
      data-exclude-search="true"
      data-exclude-hash="true"
    />
  );
}
