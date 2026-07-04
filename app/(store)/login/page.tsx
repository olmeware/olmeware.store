"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { login } from "@/lib/store";

const LoginPage = () => {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const result = login(email, password);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    router.push(result.session.role === "admin" ? "/admin" : "/");
  };

  return (
    <div className="mx-auto flex min-h-[70vh] max-w-md flex-col justify-center px-4 py-12">
      <h1 className="text-2xl font-bold tracking-tight">Log in</h1>
      <p className="mt-1 text-sm text-neutral-500">
        One login for customers and store admins.
      </p>

      <form onSubmit={handleSubmit} className="mt-8 flex flex-col gap-4">
        <label className="flex flex-col gap-1 text-sm font-medium">
          Email
          <input
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            className="rounded-lg border border-neutral-300 bg-white px-3 py-2.5 font-normal outline-none focus:border-neutral-500"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium">
          Password
          <input
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            className="rounded-lg border border-neutral-300 bg-white px-3 py-2.5 font-normal outline-none focus:border-neutral-500"
          />
        </label>
        {error && (
          <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">
            {error}
          </p>
        )}
        <button
          type="submit"
          className="rounded-lg bg-neutral-900 px-6 py-3 text-sm font-semibold text-white hover:bg-neutral-700"
        >
          Log in
        </button>
      </form>

      <p className="mt-6 text-sm text-neutral-500">
        New here?{" "}
        <Link href="/register" className="font-medium text-neutral-900 underline">
          Create an account
        </Link>
      </p>

      <div className="mt-8 rounded-xl border border-dashed border-neutral-300 bg-neutral-100 p-4 text-xs text-neutral-500">
        <p className="font-semibold text-neutral-700">Demo admin access</p>
        <p className="mt-1">
          Email <code className="font-mono">admin@olmeware.store</code> ·
          password <code className="font-mono">admin123</code>
        </p>
      </div>
    </div>
  );
};

export default LoginPage;
