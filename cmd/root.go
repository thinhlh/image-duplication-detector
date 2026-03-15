package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/imgdup/image-dupl-detector/internal/cache"
	"github.com/imgdup/image-dupl-detector/internal/comparator"
	"github.com/imgdup/image-dupl-detector/internal/config"
	"github.com/imgdup/image-dupl-detector/internal/hasher"
	"github.com/imgdup/image-dupl-detector/internal/output"
	"github.com/imgdup/image-dupl-detector/internal/prompt"
	"github.com/imgdup/image-dupl-detector/internal/scanner"
	"github.com/imgdup/image-dupl-detector/internal/startup"
	"github.com/spf13/cobra"
)

const version = "1.0.0"

var (
	flagFolder     string
	flagSimilarity int
	flagOutput     string
	flagOutFile    string
	flagRecursive  bool
	flagNoColor    bool
	flagQuiet      bool
)

// NewRootCmd builds and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "imgdup",
		Short:        "Image & Video Duplicate Detector",
		Long:         "imgdup scans a folder for visually similar images and videos above a given similarity threshold.",
		Version:      version,
		SilenceUsage: true, // don't print usage on runtime errors
		RunE:         runE,
	}

	root.Flags().StringVarP(&flagFolder, "folder", "f", "", "Folder to scan")
	root.Flags().IntVarP(&flagSimilarity, "similarity", "s", 0, "Similarity threshold 1-100 (prompts if not set)")
	root.Flags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table, json, csv")
	root.Flags().StringVar(&flagOutFile, "out-file", "", "Write results to this file path")
	root.Flags().BoolVarP(&flagRecursive, "recursive", "r", false, "Scan subfolders recursively")
	root.Flags().BoolVar(&flagNoColor, "no-color", false, "Disable ANSI colors")
	root.Flags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress progress output, print results only")

	// Register shell completion for --folder: suggest only directories
	_ = root.RegisterFlagCompletionFunc("folder", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterDirs
	})

	// Register shell completion for --output: suggest valid formats
	_ = root.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "csv"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Register shell completion for --out-file: suggest files
	_ = root.RegisterFlagCompletionFunc("out-file", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	})

	return root
}

func runE(cmd *cobra.Command, args []string) error {
	cfg := config.DefaultConfig()

	folderProvided := cmd.Flags().Changed("folder")
	similarityProvided := cmd.Flags().Changed("similarity")

	if !folderProvided {
		// Full interactive mode — no flags provided
		if err := prompt.Run(os.Stdin, cfg); err != nil {
			return err
		}
	} else {
		// Folder was given via flag — apply all flags
		cfg.Folder = flagFolder
		cfg.Recursive = flagRecursive
		cfg.NoColor = flagNoColor
		cfg.Quiet = flagQuiet
		cfg.OutputFile = flagOutFile

		switch flagOutput {
		case "json":
			cfg.OutputFormat = config.FormatJSON
		case "csv":
			cfg.OutputFormat = config.FormatCSV
		case "table", "":
			cfg.OutputFormat = config.FormatTable
		default:
			return fmt.Errorf("invalid output format %q: must be table, json, or csv", flagOutput)
		}

		if err := cfg.Validate(); err != nil {
			return err
		}

		// Prompt for similarity if -s was not explicitly set
		if !similarityProvided {
			if err := prompt.AskSimilarity(os.Stdin, cfg); err != nil {
				return err
			}
		} else {
			cfg.SimilarityPct = flagSimilarity
			if cfg.SimilarityPct < 1 || cfg.SimilarityPct > 100 {
				return fmt.Errorf("similarity must be between 1 and 100, got %d", cfg.SimilarityPct)
			}
		}
	}

	// Startup checks (ffmpeg, cache dir, folder access)
	if err := startup.Run(cfg); err != nil {
		return err
	}

	// Context with SIGINT/SIGTERM cancellation
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startTime := time.Now()

	// Open hash cache
	var c2 *cache.Cache
	if cfg.CacheEnabled {
		var err error
		c2, err = cache.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Cache unavailable: %v\n", err)
			cfg.CacheEnabled = false
		} else {
			defer c2.Close()
		}
	}

	// Count files first (for progress bar)
	if !cfg.Quiet {
		fmt.Fprint(os.Stderr, "  Counting files...\r")
	}
	total := scanner.CountFiles(ctx, cfg)
	if !cfg.Quiet {
		fmt.Fprint(os.Stderr, "                    \r")
	}

	if total == 0 {
		fmt.Fprintln(os.Stdout, "\n  No supported media files found in:", cfg.Folder)
		return nil
	}

	// Pipeline: scan → hash → compare → output
	fileCh, scanErrCh := scanner.Scan(ctx, cfg)
	hashCh, hashErrCh := hasher.Hash(ctx, fileCh, cfg, total, c2)

	go drainErrors(scanErrCh)
	go drainErrors(hashErrCh)

	var results []hasher.HashResult
	for r := range hashCh {
		results = append(results, r)
	}

	if ctx.Err() != nil {
		fmt.Fprintln(os.Stderr, "\n  ⚠ Scan interrupted.")
	}

	groups := comparator.Compare(results, cfg.SimilarityPct)

	elapsed := time.Since(startTime)
	stats := output.Stats{
		FilesScanned: len(results),
		ScanTime:     formatDuration(elapsed),
	}

	return output.Render(groups, stats, cfg)
}

func drainErrors(ch <-chan error) {
	for err := range ch {
		fmt.Fprintln(os.Stderr, err)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
