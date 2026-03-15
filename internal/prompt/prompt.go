package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/imgdup/image-dupl-detector/internal/config"
)

// Run presents the full 3-step interactive wizard and populates cfg.
func Run(in io.Reader, cfg *config.ScanConfig) error {
	sc := bufio.NewScanner(in)

	printBanner()
	fmt.Println()
	fmt.Println("  No arguments provided — starting interactive mode.")
	fmt.Println("  (Tip: run `imgdup --help` to use flags directly, with tab completion)")

	printDivider()
	fmt.Println("  Step 1 of 3 · Folder")
	if err := promptFolder(sc, cfg); err != nil {
		return err
	}

	printDivider()
	fmt.Println("  Step 2 of 3 · Similarity Threshold")
	if err := promptSimilarityBody(sc, cfg); err != nil {
		return err
	}

	printDivider()
	fmt.Println("  Step 3 of 3 · Subfolders")
	promptRecursive(sc, cfg)

	printDivider()
	fmt.Println("  Ready to scan. Here's a summary:")
	fmt.Println()
	fmt.Printf("    Folder      %s\n", cfg.Folder)
	fmt.Printf("    Similarity  %d%%\n", cfg.SimilarityPct)
	recursiveStr := "no"
	if cfg.Recursive {
		recursiveStr = "yes"
	}
	fmt.Printf("    Recursive   %s\n", recursiveStr)
	fmt.Println()
	fmt.Println("  Press Enter to start scanning, or Ctrl+C to cancel.")
	readLine(sc, "  ❯ ")
	return nil
}

// AskSimilarity prompts for only the similarity threshold (used when -f is given but -s is not).
func AskSimilarity(in io.Reader, cfg *config.ScanConfig) error {
	sc := bufio.NewScanner(in)
	printDivider()
	fmt.Println("  Similarity Threshold")
	return promptSimilarityBody(sc, cfg)
}

func promptFolder(sc *bufio.Scanner, cfg *config.ScanConfig) error {
	cwd, _ := os.Getwd()

	fmt.Println()
	fmt.Println("  Which folder should be scanned?")
	fmt.Printf("  Press Enter to use the current directory: %s\n", cwd)
	fmt.Println()
	fmt.Println("  Tip: for tab completion on the path, use the flag instead:")
	fmt.Println("       imgdup -f <Tab>   (after running: imgdup completion zsh >> ~/.zshrc)")

	for {
		input := readLine(sc, "  ❯ ")

		if input == "" {
			input = cwd
		}

		if strings.HasPrefix(input, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				printError("Cannot determine home directory: " + err.Error())
				continue
			}
			input = filepath.Join(home, input[1:])
		}

		abs, err := filepath.Abs(input)
		if err != nil {
			printError("Invalid path: " + err.Error())
			continue
		}

		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			printError("Folder not found: " + abs + "\n    Please enter a valid folder path.")
			continue
		}

		cfg.Folder = abs
		fmt.Printf("\n  ✓ Folder: %s\n", cfg.Folder)
		return nil
	}
}

func promptSimilarityBody(sc *bufio.Scanner, cfg *config.ScanConfig) error {
	fmt.Println()
	fmt.Println("  How similar should two files be to count as duplicates?")
	fmt.Println()
	fmt.Println("  ┌──────────────────────────────────────────────────────────┐")
	fmt.Println("  │  100  Exact duplicates only (pixel-perfect)              │")
	fmt.Println("  │   90  Nearly identical (recommended)                     │")
	fmt.Println("  │   80  Very similar (catches slight edits/crops)          │")
	fmt.Println("  │   70  Somewhat similar (wider net, more false positives) │")
	fmt.Println("  └──────────────────────────────────────────────────────────┘")

	for {
		fmt.Println()
		input := readLine(sc, "  ❯ Enter a number between 1 and 100: ")

		if input == "" {
			fmt.Println()
			confirm := readLine(sc, "  ❯ No value entered — use default 90%? [Y/n]: ")
			confirm = strings.ToLower(strings.TrimSpace(confirm))
			if confirm == "" || confirm == "y" || confirm == "yes" {
				cfg.SimilarityPct = 90
				fmt.Println("\n  ✓ Similarity: 90% (default)")
				return nil
			}
			continue
		}

		val, err := strconv.Atoi(input)
		if err != nil || val < 1 || val > 100 {
			printError("Please enter a whole number between 1 and 100 (e.g. 90)")
			continue
		}

		cfg.SimilarityPct = val
		fmt.Printf("\n  ✓ Similarity: %d%%\n", cfg.SimilarityPct)
		return nil
	}
}

func promptRecursive(sc *bufio.Scanner, cfg *config.ScanConfig) {
	fmt.Println()
	fmt.Printf("  Should subfolders inside %s be scanned too?\n", cfg.Folder)

	input := readLine(sc, "  ❯ [y/N]: ")
	input = strings.ToLower(strings.TrimSpace(input))

	cfg.Recursive = input == "y" || input == "yes"
	val := "no"
	if cfg.Recursive {
		val = "yes"
	}
	fmt.Printf("\n  ✓ Recursive: %s\n", val)
}

func readLine(sc *bufio.Scanner, promptText string) string {
	fmt.Print(promptText)
	sc.Scan()
	return strings.TrimSpace(sc.Text())
}

func printError(msg string) {
	fmt.Printf("\n  ✗ %s\n", msg)
}

func printBanner() {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│   imgdup · Image & Video Duplicate Detector  v1.0.0         │")
	fmt.Println("│   Find and clean up duplicate files, fast.                  │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

func printDivider() {
	fmt.Println()
	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Println()
}
