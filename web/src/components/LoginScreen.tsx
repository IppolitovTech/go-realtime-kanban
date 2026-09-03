import { useState, type SubmitEvent } from "react";
import { useAuth } from "../auth/AuthContext";
import { errorMessage } from "../api/client";

export function LoginScreen() {
  const { login, register } = useAuth();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      if (mode === "login") {
        await login(email, password);
      } else {
        await register(email, password, name);
      }
    } catch (err) {
      setError(errorMessage(err, mode === "login" ? "Failed to log in" : "Failed to register"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-[360px] flex-col px-4 py-16">
      <h1 className="mt-0 mb-6 text-center font-semibold text-zinc-950 dark:text-zinc-100">
        {mode === "login" ? "Log in" : "Create an account"}
      </h1>

      {error && <p className="mb-3 rounded-md bg-red-700/[0.12] px-3 py-2 text-red-700">{error}</p>}

      <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
        {mode === "register" && (
          <input
            type="text"
            placeholder="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            className="rounded-md border border-zinc-200 bg-white px-2.5 py-2 text-zinc-950 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
          />
        )}
        <input
          type="email"
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          className="rounded-md border border-zinc-200 bg-white px-2.5 py-2 text-zinc-950 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
        />
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={8}
          className="rounded-md border border-zinc-200 bg-white px-2.5 py-2 text-zinc-950 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
        />
        <button
          type="submit"
          disabled={submitting}
          className="cursor-pointer rounded-md bg-violet-600 px-3.5 py-2 text-sm text-white hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-violet-400 dark:hover:bg-violet-300"
        >
          {mode === "login" ? "Log in" : "Register"}
        </button>
      </form>

      <button
        type="button"
        onClick={() => {
          setError(null);
          setMode((m) => (m === "login" ? "register" : "login"));
        }}
        className="mt-4 cursor-pointer bg-transparent text-center text-sm text-violet-600 underline dark:text-violet-400"
      >
        {mode === "login" ? "Need an account? Register" : "Already have an account? Log in"}
      </button>
    </div>
  );
}
