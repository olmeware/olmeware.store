import Footer from "@/components/footer";
import Header from "@/components/header";

const StoreLayout = ({ children }: { children: React.ReactNode }) => (
  <div className="flex min-h-screen flex-col bg-neutral-50 text-neutral-900">
    <Header />
    <main className="flex-1">{children}</main>
    <Footer />
  </div>
);

export default StoreLayout;
