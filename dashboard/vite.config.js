import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const port = parseInt(env.PORT || env.VITE_PORT || "5174", 10);

  return {
    plugins: [react(), tailwindcss()],
    server: {
      host: "0.0.0.0",
      port: port,
      strictPort: true,
      hmr: {
        port: port,
      },
    },
  };
});
