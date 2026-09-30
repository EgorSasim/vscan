package query

import "strings"

// Preset is one built-in language or framework search.
// Names are the words a user can type. Expr is the query they expand to.
type Preset struct {
	Names []string
	Expr  string
}

// Presets is the built-in list, languages first, then frameworks.
func Presets() []Preset {
	return presetList
}

// LookupPreset returns the expression for a preset name.
// Matching ignores case. An unknown name returns false.
func LookupPreset(name string) (string, bool) {
	expr, ok := presetByName[strings.ToLower(name)]
	return expr, ok
}

var presetByName = map[string]string{}

func init() {
	for _, p := range presetList {
		if p.Expr == "" || len(p.Names) == 0 {
			panic("empty preset")
		}
		for _, name := range p.Names {
			key := strings.ToLower(name)
			if _, ok := presetByName[key]; ok {
				panic("duplicate preset " + name)
			}
			presetByName[key] = p.Expr
		}
	}
}

var presetList = []Preset{
	{[]string{"Go", "Golang"}, "Golang|Go"},
	{[]string{"JavaScript", "JS"}, "JavaScript|JS"},
	{[]string{"TypeScript", "TS"}, "TypeScript|TS"},
	{[]string{"Python", "Py"}, "Python"},
	{[]string{"Java"}, "Java"},
	{[]string{"Kotlin"}, "Kotlin"},
	{[]string{"Swift"}, "Swift"},
	{[]string{"Rust"}, "Rust"},
	{[]string{"PHP"}, "PHP"},
	{[]string{"Ruby"}, "Ruby"},
	{[]string{"C"}, "C"},
	{[]string{"C++", "CPP", "CPlusPlus"}, "C++|CPP|CPlusPlus"},
	{[]string{"C#", "CSharp"}, "C#|CSharp"},
	{[]string{"Scala"}, "Scala"},
	{[]string{"Elixir"}, "Elixir"},
	{[]string{"Clojure"}, "Clojure"},
	{[]string{"Dart"}, "Dart"},
	{[]string{"Haskell"}, "Haskell"},
	{[]string{"Lua"}, "Lua"},
	{[]string{"Perl"}, "Perl"},
	{[]string{"R"}, "R"},
	{[]string{"SQL"}, "SQL"},
	{[]string{"Solidity"}, "Solidity"},
	{[]string{"Erlang"}, "Erlang"},
	{[]string{"F#", "FSharp"}, "F#|FSharp"},
	{[]string{"Objective-C", "ObjC"}, "Objective-C|ObjC"},
	{[]string{"Groovy"}, "Groovy"},
	{[]string{"MATLAB"}, "MATLAB"},
	{[]string{"Zig"}, "Zig"},
	{[]string{"HTML"}, "HTML"},
	{[]string{"CSS"}, "CSS"},
	{[]string{"Bash", "Shell"}, "Bash|Shell"},
	{[]string{"PowerShell"}, "PowerShell"},

	{[]string{"Angular", "AngularJS"}, "Angular|AngularJS"},
	{[]string{"React", "ReactJS", "React.js"}, "React|ReactJS|React.js"},
	{[]string{"Vue", "VueJS", "Vue.js"}, "Vue|VueJS|Vue.js"},
	{[]string{"Svelte", "SvelteKit"}, "Svelte|SvelteKit"},
	{[]string{"Next", "NextJS", "Next.js"}, "Next.js|NextJS"},
	{[]string{"Nuxt", "NuxtJS", "Nuxt.js"}, "Nuxt|NuxtJS"},
	{[]string{"Node", "NodeJS", "Node.js"}, "Node|NodeJS|Node.js"},
	{[]string{"Express", "ExpressJS"}, "Express|ExpressJS"},
	{[]string{"Nest", "NestJS", "Nest.js"}, "NestJS|Nest.js"},
	{[]string{"Django"}, "Django"},
	{[]string{"Flask"}, "Flask"},
	{[]string{"FastAPI"}, "FastAPI"},
	{[]string{"Spring", "SpringBoot"}, "Spring|SpringBoot"},
	{[]string{"Rails", "RubyOnRails", "RoR"}, "Rails|RubyOnRails|RoR"},
	{[]string{"Laravel"}, "Laravel"},
	{[]string{"Symfony"}, "Symfony"},
	{[]string{"Flutter"}, "Flutter"},
	{[]string{"ReactNative", "React-Native"}, "ReactNative|React-Native"},
	{[]string{"DotNet", ".NET"}, "DotNet|.NET"},
	{[]string{"ASP.NET", "ASPNet"}, "ASP.NET"},
	{[]string{"Blazor"}, "Blazor"},
	{[]string{"MAUI"}, "MAUI"},
	{[]string{"WPF"}, "WPF"},
	{[]string{"WinForms"}, "WinForms"},
	{[]string{"Xamarin"}, "Xamarin"},
	{[]string{"Gin"}, "Gin"},
	{[]string{"Fiber"}, "Fiber"},
	{[]string{"Ktor"}, "Ktor"},
	{[]string{"Phoenix"}, "Phoenix"},
	{[]string{"Actix"}, "Actix"},
	{[]string{"Axum"}, "Axum"},
	{[]string{"Rocket"}, "Rocket"},
	{[]string{"Micronaut"}, "Micronaut"},
	{[]string{"Quarkus"}, "Quarkus"},
	{[]string{"Hibernate"}, "Hibernate"},
	{[]string{"JavaFX"}, "JavaFX"},
	{[]string{"Jetpack", "Compose"}, "Jetpack|Compose"},
	{[]string{"SwiftUI"}, "SwiftUI"},
	{[]string{"UIKit"}, "UIKit"},
	{[]string{"Electron"}, "Electron"},
	{[]string{"jQuery"}, "jQuery"},
	{[]string{"Redux"}, "Redux"},
	{[]string{"GraphQL"}, "GraphQL"},
	{[]string{"Tailwind"}, "Tailwind"},
	{[]string{"Bootstrap"}, "Bootstrap"},
	{[]string{"Ember"}, "Ember"},
	{[]string{"Remix"}, "Remix"},
	{[]string{"Astro"}, "Astro"},
	{[]string{"SolidJS"}, "SolidJS"},
	{[]string{"Qwik"}, "Qwik"},
	{[]string{"HTMX"}, "HTMX"},
	{[]string{"TensorFlow"}, "TensorFlow"},
	{[]string{"PyTorch"}, "PyTorch"},
	{[]string{"Android"}, "Android"},
	{[]string{"iOS"}, "iOS"},
	{[]string{"Unity"}, "Unity"},
	{[]string{"Unreal"}, "Unreal"},
}
