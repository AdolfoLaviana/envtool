package main

import (
	"fmt"
	"os"

	"github.com/adolfoc/envtool/internal/audit"
	"github.com/adolfoc/envtool/internal/crypto"
	"github.com/adolfoc/envtool/internal/differ"
	"github.com/adolfoc/envtool/internal/generator"
	"github.com/adolfoc/envtool/internal/linter"
	"github.com/adolfoc/envtool/internal/sync"
	"github.com/adolfoc/envtool/internal/validator"
)

func printUsage() {
	fmt.Println("EnvTool - Environment File Management Tool")
	fmt.Println("Usage:")
	fmt.Println("  envtool validate <file>            - Validate .env file format and contents")
	fmt.Println("  envtool diff <file1> <file2>       - Compare two .env files")
	fmt.Println("  envtool encrypt <file> --out <out> - Encrypt .env file")
	fmt.Println("  envtool decrypt <file> --out <out> - Decrypt .env file")
	fmt.Println("  envtool push <file> --env <env>    - Upload .env file to remote storage")
	fmt.Println("  envtool pull --env <env>           - Download .env file from remote storage")
	fmt.Println("  envtool generate --schema <schema> - Generate .env file from schema")
	fmt.Println("  envtool audit <file>               - Show change history for .env file")
	fmt.Println("  envtool lint <file> [--fix]        - Lint .env file and optionally fix issues")
	fmt.Println("  envtool help                       - Show this help message")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "validate":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing file argument")
			return
		}
		validator.ValidateEnv(os.Args[2])
	case "diff":
		if len(os.Args) < 4 {
			fmt.Println("Error: Missing file arguments")
			return
		}
		differ.DiffEnvFiles(os.Args[2], os.Args[3])
	case "encrypt":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing file argument")
			return
		}
		outFile := ""
		for i, arg := range os.Args {
			if arg == "--out" && i+1 < len(os.Args) {
				outFile = os.Args[i+1]
				break
			}
		}
		crypto.EncryptEnv(os.Args[2], outFile)
	case "decrypt":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing file argument")
			return
		}
		outFile := ""
		for i, arg := range os.Args {
			if arg == "--out" && i+1 < len(os.Args) {
				outFile = os.Args[i+1]
				break
			}
		}
		crypto.DecryptEnv(os.Args[2], outFile)
	case "push":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing file argument")
			return
		}
		env := "production"
		for i, arg := range os.Args {
			if arg == "--env" && i+1 < len(os.Args) {
				env = os.Args[i+1]
				break
			}
		}
		sync.PushEnv(os.Args[2], env)
	case "pull":
		env := "staging"
		for i, arg := range os.Args {
			if arg == "--env" && i+1 < len(os.Args) {
				env = os.Args[i+1]
				break
			}
		}
		sync.PullEnv(env)
	case "generate":
		schema := ""
		for i, arg := range os.Args {
			if arg == "--schema" && i+1 < len(os.Args) {
				schema = os.Args[i+1]
				break
			}
		}
		generator.GenerateEnv(schema)
	case "audit":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing file argument")
			return
		}
		audit.AuditEnv(os.Args[2])
	case "lint":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing file argument")
			return
		}
		fix := false
		for _, arg := range os.Args {
			if arg == "--fix" {
				fix = true
				break
			}
		}
		linter.LintEnv(os.Args[2], fix)
	case "help":
		printUsage()
	default:
		fmt.Printf("Error: Unknown command '%s'\n", command)
		printUsage()
	}
}
