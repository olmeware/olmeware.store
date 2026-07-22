package seed

// seedCollection mirrors a storefront collection.
type seedCollection struct {
	Slug, Name, Description string
	SortOrder               int
}

// seedProduct mirrors lib/seed.ts. Prices are in MXN major units and converted
// to minor units (×100) on insert. Sizes become product_variants.
type seedProduct struct {
	Slug        string
	Name        string
	Description string
	Garment     string // shirt | sweater | hoodie | cap
	Stack       string
	Tech        string
	LogoSlug    string
	PriceMajor  int
	ColorHex    string
	Sizes       []string
	Collection  string // collection slug, or "" for none
	Featured    bool
}

var seedCollections = []seedCollection{
	{Slug: "new-arrivals", Name: "New Arrivals", Description: "The latest drops, fresh from the print shop.", SortOrder: 0},
	{Slug: "classics", Name: "Classics", Description: "Timeless stacks that never go out of style.", SortOrder: 1},
	{Slug: "ai-drop", Name: "AI Drop", Description: "Wear the models that changed everything.", SortOrder: 2},
}

const (
	black = "#1a1a1a"
	white = "#f5f5f5"
)

var seedProducts = []seedProduct{
	{"python-classic-tee", "Python Classic Tee", "Soft cotton tee with the Python logo front and center. Indentation not included.", "shirt", "languages", "Python", "python", 449, black, []string{"S", "M", "L", "XL", "XXL"}, "classics", true},
	{"typescript-strict-tee", "TypeScript Strict Tee", "For those who never use any. Blue square, big energy, fully typed comfort.", "shirt", "languages", "TypeScript", "typescript", 449, white, []string{"XS", "S", "M", "L", "XL"}, "classics", true},
	{"javascript-og-tee", "JavaScript OG Tee", "The language that runs the web, on the shirt that runs your wardrobe.", "shirt", "languages", "JavaScript", "javascript", 429, black, []string{"S", "M", "L", "XL"}, "classics", false},
	{"react-atomic-hoodie", "React Atomic Hoodie", "Heavyweight hoodie with the atom everyone re-renders for. Hooks sold separately.", "hoodie", "frontend", "React", "react", 899, black, []string{"S", "M", "L", "XL", "XXL"}, "new-arrivals", true},
	{"vue-progressive-tee", "Vue Progressive Tee", "Approachable, performant, versatile. Also, it's green.", "shirt", "frontend", "Vue", "vuejs", 429, white, []string{"S", "M", "L", "XL"}, "", false},
	{"tailwind-utility-sweater", "Tailwind Utility Sweater", "flex items-center justify-cozy. A utility-first sweater for utility-first people.", "sweater", "frontend", "Tailwind CSS", "tailwindcss", 749, black, []string{"S", "M", "L", "XL"}, "new-arrivals", false},
	{"nextjs-edge-hoodie", "Next.js Edge Hoodie", "Server-rendered warmth with zero layout shift. Ships instantly to your closet.", "hoodie", "frontend", "Next.js", "nextjs", 949, black, []string{"S", "M", "L", "XL", "XXL"}, "new-arrivals", true},
	{"nodejs-runtime-tee", "Node.js Runtime Tee", "Non-blocking, event-driven, machine-washable.", "shirt", "backend", "Node.js", "nodejs", 449, black, []string{"S", "M", "L", "XL"}, "", false},
	{"go-gopher-tee", "Go Gopher Tee", "Concurrency you can wear. Goroutines not included, gopher is.", "shirt", "backend", "Go", "go", 449, white, []string{"S", "M", "L", "XL", "XXL"}, "", false},
	{"rust-fearless-hoodie", "Rust Fearless Hoodie", "Memory-safe warmth with zero-cost abstractions. The borrow checker approves.", "hoodie", "languages", "Rust", "rust", 899, black, []string{"S", "M", "L", "XL"}, "classics", false},
	{"docker-container-cap", "Docker Container Cap", "Works on your head. Works on every head. That's the point.", "cap", "devops", "Docker", "docker", 349, black, []string{"M", "L"}, "", false},
	{"kubernetes-helm-sweater", "Kubernetes Helm Sweater", "Self-healing comfort that scales with you. Warmth orchestrated across all pods.", "sweater", "devops", "Kubernetes", "kubernetes", 779, black, []string{"S", "M", "L", "XL"}, "", false},
	{"git-commit-cap", "Git Commit Cap", "git checkout style. Merge conflicts with your outfit resolved.", "cap", "tools", "Git", "git", 329, black, []string{"M", "L"}, "", false},
	{"linux-tux-tee", "Linux Tux Tee", "The penguin that powers the internet, now powering your look.", "shirt", "tools", "Linux", "linux", 449, white, []string{"S", "M", "L", "XL", "XXL"}, "", false},
	{"pytorch-gradient-hoodie", "PyTorch Gradient Hoodie", "Backpropagate warmth through every layer. Autograd for your torso.", "hoodie", "ai-ml", "PyTorch", "pytorch", 929, black, []string{"S", "M", "L", "XL"}, "ai-drop", true},
	{"graphql-query-tee", "GraphQL Query Tee", "Ask for exactly what you want. Get exactly this shirt.", "shirt", "backend", "GraphQL", "graphql", 449, white, []string{"S", "M", "L", "XL"}, "new-arrivals", false},
}
