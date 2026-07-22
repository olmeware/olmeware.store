"use client";

import { useMemo, useState } from "react";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  PaymentElement,
  useElements,
  useStripe,
} from "@stripe/react-stripe-js";

type Props = {
  publishableKey: string;
  clientSecret: string;
  total: string;
  onSuccess: () => void;
};

// StripePayment renders the Stripe Payment Element and confirms the card
// payment in place (no redirect for card methods).
const StripePayment = ({ publishableKey, clientSecret, total, onSuccess }: Props) => {
  const stripePromise = useMemo(() => loadStripe(publishableKey), [publishableKey]);
  return (
    <Elements
      stripe={stripePromise}
      options={{ clientSecret, appearance: { theme: "stripe" } }}
    >
      <PaymentForm total={total} onSuccess={onSuccess} />
    </Elements>
  );
};

const PaymentForm = ({ total, onSuccess }: { total: string; onSuccess: () => void }) => {
  const stripe = useStripe();
  const elements = useElements();
  const [error, setError] = useState("");
  const [paying, setPaying] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!stripe || !elements) return;
    setPaying(true);
    setError("");
    const { error: err } = await stripe.confirmPayment({
      elements,
      redirect: "if_required",
    });
    if (err) {
      setError(err.message ?? "Payment could not be completed.");
      setPaying(false);
      return;
    }
    onSuccess();
  };

  return (
    <form onSubmit={submit} className="mt-6 flex flex-col gap-4">
      <PaymentElement />
      {error && (
        <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{error}</p>
      )}
      <button
        type="submit"
        disabled={!stripe || paying}
        className="w-full rounded-lg bg-neutral-900 px-6 py-3 text-sm font-semibold text-white hover:bg-neutral-700 disabled:opacity-60"
      >
        {paying ? "Processing…" : `Pay ${total}`}
      </button>
      <p className="text-center text-xs text-neutral-400">
        Test mode · use card 4242 4242 4242 4242, any future date & CVC.
      </p>
    </form>
  );
};

export default StripePayment;
