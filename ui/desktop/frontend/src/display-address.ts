export function displayListenAddress(address?: string) {
  return (address ?? "").replace(/^(?:0\.0\.0\.0|\[::\])(?=:)/, "127.0.0.1");
}
