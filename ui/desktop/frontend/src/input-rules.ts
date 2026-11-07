                                                                                      
export function sourceAddressError(value: string) {
  const address = value.trim();
  if (!address) return "";
  if (/^(experience|realm|gathering):\S.*$/i.test(address)) return "";
  let raw = address;
  if (!raw.includes("://")) {
    if (/^[\da-f:]+$/i.test(raw) && raw.split(":").length > 2) raw = `[${raw}]`;
    raw = "http://" + raw;
  }
  try {
    const url = new URL(raw);
    if (!["http:", "https:"].includes(url.protocol))
      return "Use an HTTP or HTTPS endpoint.";
    if (url.username || url.password)
      return "Remove the username and password.";
    if (url.pathname !== "/" || url.search || url.hash)
      return "Use a server address without a path, query or fragment.";
    if (!url.hostname || /\s/.test(url.hostname))
      return "Enter a host or IP address.";
    if (url.port && (+url.port < 1 || +url.port > 65535))
      return "Use a port from 1 to 65535.";
    return "";
  } catch {
    return "Enter a host or IP with an optional port from 1 to 65535.";
  }
}
export function nonBlankName(value: string) {
  return value.trim() ? "" : "Enter a name.";
}
export function profileNameError(value: string) {
  const name = value.trim();
  if (name === "." || name === ".." || /[<>:"/\\|?*\x00-\x1f]/.test(name))
    return "Use a profile name without path or special characters.";
  return "";
}
export function rangeOrderError(
  minimum: number | string,
  maximum: number | string,
) {
  if (
    String(minimum).trim() &&
    String(maximum).trim() &&
    Number(minimum) > Number(maximum)
  )
    return "Minimum must not exceed maximum.";
  return "";
}
export function fileNameError(value: string) {
  if (!value.trim()) return "";
  if (/[<>:"/\\|?*\x00-\x1f]/.test(value))
    return 'Remove these characters: < > : " / \\ | ? *.';
  if (/[. ]$/.test(value)) return "The name cannot end with a dot or space.";
  if (/^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\.|$)/i.test(value))
    return "Choose a name that is not reserved by Windows.";
  return "";
}
export function decimalError(value: string, min: number, max: number) {
  const n = Number(value.replace(",", "."));
  if (!value.trim() || !Number.isFinite(n)) return "Enter a valid number.";
  if (n < min || n > max) return `Enter a value from ${min} to ${max}.`;
  return "";
}
