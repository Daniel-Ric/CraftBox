export default function StudioBrand({ size = 28 }: { size?: number }) {
  return (
    <img
      className="studio-brand-mark"
      src="/craft-box-icon.png"
      width={size}
      height={size}
      alt=""
    />
  );
}
