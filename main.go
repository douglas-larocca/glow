// Package main provides the entry point for the Glow CLI application.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/glow/v2/ui"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	gap "github.com/muesli/go-app-paths"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var (
	// Version as provided by goreleaser.
	Version = ""
	// CommitSHA as provided by goreleaser.
	CommitSHA = ""

	readmeNames      = []string{"README.md", "README", "Readme.md", "Readme", "readme.md", "readme"}
	configFile       string
	pager            bool
	tui              bool
	style            string
	width            uint
	showAllFiles     bool
	showLineNumbers  bool
	preserveNewLines bool
	mouse            bool

	rootCmd = &cobra.Command{
		Use:   "glow [SOURCE|DIR]",
		Short: "Render markdown on the CLI, with pizzazz!",
		Long: paragraph(
			fmt.Sprintf("\nRender markdown on the CLI, %s!", keyword("with pizzazz")),
		),
		SilenceErrors:    false,
		SilenceUsage:     true,
		TraverseChildren: true,
		Args:             cobra.MaximumNArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return validateOptions(cmd)
		},
		RunE: execute,
	}
)

// source provides a readable markdown source.
type source struct {
	reader io.ReadCloser
	URL    string
}

// sourceFromArg parses an argument and creates a readable source for it.
func sourceFromArg(arg string) (*source, error) {
	// from stdin
	if arg == "-" {
		return &source{reader: os.Stdin}, nil
	}

	// a GitHub or GitLab URL (even without the protocol):
	src, err := readmeURL(arg)
	if src != nil && err == nil {
		// if there's an error, try next methods...
		return src, nil
	}

	// HTTP(S) URLs:
	if u, err := url.ParseRequestURI(arg); err == nil && strings.Contains(arg, "://") { //nolint:nestif
		if u.Scheme != "" {
			if u.Scheme != "http" && u.Scheme != "https" {
				return nil, fmt.Errorf("%s is not a supported protocol", u.Scheme)
			}
			// consumer of the source is responsible for closing the ReadCloser.
			resp, err := http.Get(u.String()) //nolint: noctx,bodyclose
			if err != nil {
				return nil, fmt.Errorf("unable to get url: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
			}
			return &source{resp.Body, u.String()}, nil
		}
	}

	// a directory:
	if len(arg) == 0 {
		// use the current working dir if no argument was supplied
		arg = "."
	}
	st, err := os.Stat(arg)
	if err == nil && st.IsDir() { //nolint:nestif
		var src *source
		_ = filepath.Walk(arg, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			for _, v := range readmeNames {
				if strings.EqualFold(filepath.Base(path), v) {
					r, err := os.Open(path)
					if err != nil {
						continue
					}

					u, _ := filepath.Abs(path)
					src = &source{r, u}

					// abort filepath.Walk
					return errors.New("source found")
				}
			}
			return nil
		})

		if src != nil {
			return src, nil
		}

		return nil, errors.New("missing markdown source")
	}

	r, err := os.Open(arg)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	u, err := filepath.Abs(arg)
	if err != nil {
		return nil, fmt.Errorf("unable to get absolute path: %w", err)
	}
	return &source{r, u}, nil
}

// validateStyle checks if the style is a default style, if not, checks that
// the custom style exists.
func validateStyle(style string) error {
	if style != "auto" && styles.DefaultStyles[style] == nil {
		style = utils.ExpandPath(style)
		if _, err := os.Stat(style); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("specified style does not exist: %s", style)
		} else if err != nil {
			return fmt.Errorf("unable to stat file: %w", err)
		}
	}
	return nil
}

func validateOptions(cmd *cobra.Command) error {
	// grab config values from Viper
	width = viper.GetUint("width")
	mouse = viper.GetBool("mouse")
	pager = viper.GetBool("pager")
	tui = viper.GetBool("tui")
	showAllFiles = viper.GetBool("all")
	preserveNewLines = viper.GetBool("preserveNewLines")

	if pager && tui {
		return errors.New("cannot use both pager and tui")
	}

	// validate the glamour style
	style = viper.GetString("style")
	if err := validateStyle(style); err != nil {
		return err
	}

	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	// We want to use a special no-TTY style, when stdout is not a terminal
	// and there was no specific style passed by arg
	if !isTerminal && !cmd.Flags().Changed("style") {
		style = "notty"
	}

	// Detect terminal width
	if !cmd.Flags().Changed("width") { //nolint:nestif
		if isTerminal && width == 0 {
			w, _, err := term.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				width = uint(w) //nolint:gosec
			}

			if width > 120 {
				width = 120
			}
		}
		if width == 0 {
			width = 80
		}
	}
	return nil
}

func stdinIsPipe() (bool, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false, fmt.Errorf("unable to open file: %w", err)
	}
	if stat.Mode()&os.ModeCharDevice == 0 || stat.Size() > 0 {
		return true, nil
	}
	return false, nil
}

func execute(cmd *cobra.Command, args []string) error {
	// if stdin is a pipe then use stdin for input. note that you can also
	// explicitly use a - to read from stdin.
	if yes, err := stdinIsPipe(); err != nil {
		return err
	} else if yes {
		src := &source{reader: os.Stdin}
		defer src.reader.Close() //nolint:errcheck
		return executeCLI(cmd, src, os.Stdout)
	}

	switch len(args) {
	// TUI running on cwd
	case 0:
		return runTUI("", "")

	// TUI with possible dir argument
	case 1:
		// Validate that the argument is a directory. If it's not treat it as
		// an argument to the non-TUI version of Glow (via fallthrough).
		info, err := os.Stat(args[0])
		if err == nil && info.IsDir() {
			p, err := filepath.Abs(args[0])
			if err == nil {
				return runTUI(p, "")
			}
		}
		fallthrough

	// CLI
	default:
		for _, arg := range args {
			if err := executeArg(cmd, arg, os.Stdout); err != nil {
				return err
			}
		}
	}

	return nil
}

func executeArg(cmd *cobra.Command, arg string, w io.Writer) error {
	// create an io.Reader from the markdown source in cli-args
	src, err := sourceFromArg(arg)
	if err != nil {
		return err
	}
	defer src.reader.Close() //nolint:errcheck
	return executeCLI(cmd, src, w)
}

// shouldRenderUpdate determines if we should re-render based on the current line
// and content seen so far. This helps identify markdown elements that can change
// the rendering of previous content.
func shouldRenderUpdate(currentLine string, previousLines []string) bool {
	// Always render at least every 10 lines to ensure responsiveness
	if len(previousLines)%10 == 0 {
		return true
	}

	// Check for constructs that can affect previous content rendering
	patterns := []struct {
		regex *regexp.Regexp
		desc  string
	}{
		{regexp.MustCompile(`^\[.*?\]:\s+`), "reference link"},
		{regexp.MustCompile(`^\[\^.*?\]:\s+`), "footnote definition"},
		{regexp.MustCompile(`^<!--`), "HTML comment start"},
		{regexp.MustCompile(`-->`), "HTML comment end"},
		{regexp.MustCompile(`^#{1,6}\s+`), "heading"},
		{regexp.MustCompile(`^(\*\s*){3,}`), "horizontal rule"},
		{regexp.MustCompile(`^(\-\s*){3,}`), "horizontal rule"},
		{regexp.MustCompile(`^(\_\s*){3,}`), "horizontal rule"},
		{regexp.MustCompile(`^:::.*`), "fenced div start/end"},
		{regexp.MustCompile(`^\|.*\|`), "table line"},
		{regexp.MustCompile(`^(\s*\*\s+|\s*\d+\.\s+|\s*-\s+)`), "list item"},
	}

	for _, pattern := range patterns {
		if pattern.regex.MatchString(strings.TrimSpace(currentLine)) {
			return true
		}
	}

	// Check for the end of a code block which could affect rendering
	if strings.TrimSpace(currentLine) == "```" {
		// Look back to find if this is the end of a code block
		for i := len(previousLines) - 2; i >= 0; i-- {
			if strings.TrimSpace(previousLines[i]) == "```" {
				return false // This is a nested ``` within a code block
			}
			if strings.HasPrefix(strings.TrimSpace(previousLines[i]), "```") {
				return true // This is the end of a code block
			}
		}
		return true // Assume it's the end of a code block if we can't determine
	}

	// Check for completion of a complex structure
	if len(previousLines) >= 2 {
		prevLine := strings.TrimSpace(previousLines[len(previousLines)-2])
		// If we just completed a multi-line construct like a table
		if (prevLine == "" && strings.HasPrefix(currentLine, "|")) ||
			(strings.HasPrefix(prevLine, "|") && currentLine == "") {
			return true
		}
	}

	return false
}

func executeCLI(cmd *cobra.Command, src *source, w io.Writer) error {
	// If not reading from stdin, just read all and render once
	if _, ok := src.reader.(*os.File); !ok || src.reader != os.Stdin {
		b, err := io.ReadAll(src.reader)
		if err != nil {
			return fmt.Errorf("unable to read from reader: %w", err)
		}
		return renderMarkdown(cmd, src, b, w)
	}

	// For stdin, check if it's a terminal or a pipe
	if file, ok := src.reader.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		// If stdin is a terminal and not a pipe, just read all at once
		b, err := io.ReadAll(src.reader)
		if err != nil {
			return fmt.Errorf("unable to read from reader: %w", err)
		}
		return renderMarkdown(cmd, src, b, w)
	}

	// For stdin from a pipe, we'll read incrementally and render as we go
	return renderIncrementalFromStdin(cmd, src, w)
}

// renderIncrementalFromStdin reads incrementally from stdin and renders
// the markdown as it becomes available
func renderIncrementalFromStdin(cmd *cobra.Command, src *source, w io.Writer) error {
	scanner := bufio.NewScanner(src.reader)

	// Buffer to accumulate content
	var buffer bytes.Buffer
	var previousLines []string // Store individual lines for diffing
	var lastOutput string      // Last output sent to terminal
	var r *glamour.TermRenderer
	var err error

	// Setup renderer once
	if r, _, err = setupRenderer(src); err != nil {
		return err
	}

	for scanner.Scan() {
		// Get the new line
		line := scanner.Text()

		// Add the line to our accumulated content
		buffer.WriteString(line)
		buffer.WriteString("\n")

		// Add to our line-by-line tracking
		previousLines = append(previousLines, line)

		// Only re-render periodically or when we detect certain markdown structures
		// that might affect previous rendering
		shouldRender := shouldRenderUpdate(line, previousLines)

		if shouldRender {
			// Generate new full output
			newOutput, err := renderContentIncremental(r, src, buffer.Bytes(), "")
			if err != nil {
				return err
			}

			// If rendering drastically changed (can happen with reference links, footnotes, etc.)
			if !strings.HasPrefix(newOutput, lastOutput) {
				// Clear terminal and do a full re-render if the output doesn't just append
				fmt.Fprint(w, "\033[2J\033[H")
				if _, err = fmt.Fprint(w, newOutput); err != nil {
					return fmt.Errorf("unable to write to writer: %w", err)
				}
			} else {
				// Get only the new part of the rendered output
				newContent := strings.TrimPrefix(newOutput, lastOutput)
				if _, err = fmt.Fprint(w, newContent); err != nil {
					return fmt.Errorf("unable to write to writer: %w", err)
				}
			}

			lastOutput = newOutput
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	// Ensure final render happens if we stopped early due to periodic rendering
	newOutput, err := renderContentIncremental(r, src, buffer.Bytes(), "")
	if err != nil {
		return err
	}

	if newOutput != lastOutput {
		if !strings.HasPrefix(newOutput, lastOutput) {
			// Full re-render
			fmt.Fprint(w, "\033[2J\033[H")
			if _, err = fmt.Fprint(w, newOutput); err != nil {
				return fmt.Errorf("unable to write to writer: %w", err)
			}
		} else {
			// Incremental update
			newContent := strings.TrimPrefix(newOutput, lastOutput)
			if _, err = fmt.Fprint(w, newContent); err != nil {
				return fmt.Errorf("unable to write to writer: %w", err)
			}
		}
	}

	return nil
}

// setupRenderer creates a glamour renderer with proper configuration
func setupRenderer(src *source) (*glamour.TermRenderer, string, error) {
	var baseURL string
	u, err := url.ParseRequestURI(src.URL)
	if err == nil {
		u.Path = filepath.Dir(u.Path)
		baseURL = u.String() + "/"
	}

	isCode := !utils.IsMarkdownFile(src.URL)

	// Initialize glamour
	r, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		utils.GlamourStyle(style, isCode),
		glamour.WithWordWrap(int(width)),
		glamour.WithBaseURL(baseURL),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return nil, "", fmt.Errorf("unable to create renderer: %w", err)
	}

	return r, baseURL, nil
}

// renderContentIncremental renders the provided markdown content and returns the rendered output
// This is used for incremental rendering to compare with previous output
func renderContentIncremental(r *glamour.TermRenderer, src *source, content []byte, lastOutput string) (string, error) {
	// Apply frontmatter removal
	contentWithoutFrontmatter := utils.RemoveFrontmatter(content)

	// Handle code files
	contentStr := string(contentWithoutFrontmatter)
	isCode := !utils.IsMarkdownFile(src.URL)
	if isCode {
		contentStr = utils.WrapCodeBlock(contentStr, filepath.Ext(src.URL))
	}

	// Render the content
	out, err := r.Render(contentStr)
	if err != nil {
		return "", fmt.Errorf("unable to render markdown: %w", err)
	}

	return out, nil
}

// renderContent renders the provided markdown content to the writer
// This is used for one-time full rendering
func renderContent(r *glamour.TermRenderer, src *source, content []byte, w io.Writer) error {
	out, err := renderContentIncremental(r, src, content, "")
	if err != nil {
		return err
	}

	// Output the rendered content
	if _, err = fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("unable to write to writer: %w", err)
	}

	return nil
}

// renderMarkdown handles the one-time rendering of markdown content (non-stdin case)
func renderMarkdown(cmd *cobra.Command, src *source, content []byte, w io.Writer) error {
	content = utils.RemoveFrontmatter(content)

	// Setup renderer
	r, _, err := setupRenderer(src)
	if err != nil {
		return err
	}

	// Render
	contentStr := string(content)
	isCode := !utils.IsMarkdownFile(src.URL)
	if isCode {
		contentStr = utils.WrapCodeBlock(contentStr, filepath.Ext(src.URL))
	}

	out, err := r.Render(contentStr)
	if err != nil {
		return fmt.Errorf("unable to render markdown: %w", err)
	}

	// Display
	switch {
	case pager || cmd.Flags().Changed("pager"):
		pagerCmd := os.Getenv("PAGER")
		if pagerCmd == "" {
			pagerCmd = "less -r"
		}

		pa := strings.Split(pagerCmd, " ")
		c := exec.Command(pa[0], pa[1:]...)
		c.Stdin = strings.NewReader(out)
		c.Stdout = os.Stdout
		if err := c.Run(); err != nil {
			return fmt.Errorf("unable to run command: %w", err)
		}
		return nil
	case tui || cmd.Flags().Changed("tui"):
		path := ""
		if !isURL(src.URL) {
			path = src.URL
		}
		return runTUI(path, contentStr)
	default:
		if _, err = fmt.Fprint(w, out); err != nil {
			return fmt.Errorf("unable to write to writer: %w", err)
		}
		return nil
	}
}

func runTUI(path string, content string) error {
	// Read environment to get debugging stuff
	cfg, err := env.ParseAs[ui.Config]()
	if err != nil {
		return fmt.Errorf("error parsing config: %v", err)
	}

	// use style set in env, or auto if unset
	if err := validateStyle(cfg.GlamourStyle); err != nil {
		cfg.GlamourStyle = style
	}

	cfg.Path = path
	cfg.ShowAllFiles = showAllFiles
	cfg.ShowLineNumbers = showLineNumbers
	cfg.GlamourMaxWidth = width
	cfg.EnableMouse = mouse
	cfg.PreserveNewLines = preserveNewLines

	// Run Bubble Tea program
	if _, err := ui.NewProgram(cfg, content).Run(); err != nil {
		return fmt.Errorf("unable to run tui program: %w", err)
	}

	return nil
}

func main() {
	closer, err := setupLog()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		_ = closer()
		os.Exit(1)
	}
	_ = closer()
}

func init() {
	tryLoadConfigFromDefaultPlaces()
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		Version = "unknown (built from source)"
	}
	rootCmd.Version = Version
	rootCmd.InitDefaultCompletionCmd()

	// "Glow Classic" cli arguments
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", fmt.Sprintf("config file (default %s)", viper.GetViper().ConfigFileUsed()))
	rootCmd.Flags().BoolVarP(&pager, "pager", "p", false, "display with pager")
	rootCmd.Flags().BoolVarP(&tui, "tui", "t", false, "display with tui")
	rootCmd.Flags().StringVarP(&style, "style", "s", styles.AutoStyle, "style name or JSON path")
	rootCmd.Flags().UintVarP(&width, "width", "w", 0, "word-wrap at width (set to 0 to disable)")
	rootCmd.Flags().BoolVarP(&showAllFiles, "all", "a", false, "show system files and directories (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&showLineNumbers, "line-numbers", "l", false, "show line numbers (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&preserveNewLines, "preserve-new-lines", "n", false, "preserve newlines in the output")
	rootCmd.Flags().BoolVarP(&mouse, "mouse", "m", false, "enable mouse wheel (TUI-mode only)")
	_ = rootCmd.Flags().MarkHidden("mouse")

	// Config bindings
	_ = viper.BindPFlag("pager", rootCmd.Flags().Lookup("pager"))
	_ = viper.BindPFlag("tui", rootCmd.Flags().Lookup("tui"))
	_ = viper.BindPFlag("style", rootCmd.Flags().Lookup("style"))
	_ = viper.BindPFlag("width", rootCmd.Flags().Lookup("width"))
	_ = viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	_ = viper.BindPFlag("mouse", rootCmd.Flags().Lookup("mouse"))
	_ = viper.BindPFlag("preserveNewLines", rootCmd.Flags().Lookup("preserve-new-lines"))
	_ = viper.BindPFlag("showLineNumbers", rootCmd.Flags().Lookup("line-numbers"))
	_ = viper.BindPFlag("all", rootCmd.Flags().Lookup("all"))

	viper.SetDefault("style", styles.AutoStyle)
	viper.SetDefault("width", 0)
	viper.SetDefault("all", true)

	rootCmd.AddCommand(configCmd, manCmd)
}

func tryLoadConfigFromDefaultPlaces() {
	scope := gap.NewScope(gap.User, "glow")
	dirs, err := scope.ConfigDirs()
	if err != nil {
		fmt.Println("Could not load find configuration directory.")
		os.Exit(1)
	}

	if c := os.Getenv("XDG_CONFIG_HOME"); c != "" {
		dirs = append([]string{filepath.Join(c, "glow")}, dirs...)
	}

	if c := os.Getenv("GLOW_CONFIG_HOME"); c != "" {
		dirs = append([]string{c}, dirs...)
	}

	for _, v := range dirs {
		viper.AddConfigPath(v)
	}

	viper.SetConfigName("glow")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("glow")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Warn("Could not parse configuration file", "err", err)
		}
	}

	if used := viper.ConfigFileUsed(); used != "" {
		log.Debug("Using configuration file", "path", viper.ConfigFileUsed())
		return
	}

	if viper.ConfigFileUsed() == "" {
		configFile = filepath.Join(dirs[0], "glow.yml")
	}
	if err := ensureConfigFile(); err != nil {
		log.Error("Could not create default configuration", "error", err)
	}
}
