package browser

import (
	"context"
	"fmt"

	"github.com/ai-agent-framework/pkg/interfaces"
	"github.com/mxschmitt/playwright-go"
)

// PlaywrightAgent implements the BrowserAgent interface using Playwright
type PlaywrightAgent struct {
	browser  playwright.Browser
	page     playwright.Page
	logger   interfaces.Logger
	headless bool
}

// NewPlaywrightAgent creates a new Playwright-based browser agent
func NewPlaywrightAgent(logger interfaces.Logger, headless bool) *PlaywrightAgent {
	return &PlaywrightAgent{
		logger:   logger,
		headless: headless,
	}
}

// Initialize starts the browser and creates a new page
func (p *PlaywrightAgent) Initialize(ctx context.Context) error {
	p.logger.Info("Initializing Playwright browser")

	// Install Playwright browsers if needed
	err := playwright.Install()
	if err != nil {
		return fmt.Errorf("failed to install Playwright: %w", err)
	}

	// Launch Playwright
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("failed to start Playwright: %w", err)
	}

	// Launch browser
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(p.headless),
	})
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	p.browser = browser

	// Create new page
	page, err := browser.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create new page: %w", err)
	}

	p.page = page

	p.logger.WithField("headless", p.headless).Info("Playwright browser initialized")
	return nil
}

// Navigate navigates to the specified URL
func (p *PlaywrightAgent) Navigate(ctx context.Context, url string) error {
	p.logger.WithField("url", url).Info("Navigating to URL")

	if p.page == nil {
		return fmt.Errorf("browser not initialized")
	}

	_, err := p.page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000), // 30 second timeout
	})
	if err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// Additional wait to ensure the page is fully interactive
	p.page.WaitForLoadState("networkidle")

	// Wait a bit more for JavaScript to initialize
	p.page.WaitForTimeout(1000) // 1 second

	p.logger.WithField("url", url).Info("Navigation completed")
	return nil
}

// ExecuteAction performs a browser action based on the action type
func (p *PlaywrightAgent) ExecuteAction(ctx context.Context, action interfaces.BrowserAction) (interface{}, error) {
	p.logger.WithFields(map[string]interface{}{
		"action_type": action.Type,
		"selector":    action.Selector,
	}).Info("Executing browser action")

	if p.page == nil {
		return nil, fmt.Errorf("browser not initialized")
	}

	switch action.Type {
	case "click":
		return p.handleClick(action)
	case "type":
		return p.handleType(action)
	case "select":
		return p.handleSelect(action)
	case "wait":
		return p.handleWait(action)
	case "scroll":
		return p.handleScroll(action)
	case "extract_text":
		return p.handleExtractText(action)
	case "extract_attribute":
		return p.handleExtractAttribute(action)
	default:
		return nil, fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// Screenshot takes a screenshot of the current page
func (p *PlaywrightAgent) Screenshot(ctx context.Context) ([]byte, error) {
	p.logger.Info("Taking screenshot")

	if p.page == nil {
		return nil, fmt.Errorf("browser not initialized")
	}

	screenshot, err := p.page.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to take screenshot: %w", err)
	}

	p.logger.Info("Screenshot taken successfully")
	return screenshot, nil
}

// GetPageContent returns the HTML content of the current page
func (p *PlaywrightAgent) GetPageContent(ctx context.Context) (string, error) {
	p.logger.Info("Getting page content")

	if p.page == nil {
		return "", fmt.Errorf("browser not initialized")
	}

	content, err := p.page.Content()
	if err != nil {
		return "", fmt.Errorf("failed to get page content: %w", err)
	}

	p.logger.WithField("content_length", len(content)).Info("Page content retrieved")
	return content, nil
}

// Close closes the browser and cleans up resources
func (p *PlaywrightAgent) Close(ctx context.Context) error {
	p.logger.Info("Closing browser")

	if p.browser != nil {
		if err := p.browser.Close(); err != nil {
			p.logger.WithField("error", err).Error("Failed to close browser")
			return err
		}
	}

	p.logger.Info("Browser closed successfully")
	return nil
}

// Action handlers

func (p *PlaywrightAgent) handleClick(action interfaces.BrowserAction) (interface{}, error) {
	// Wait for the element to be available and visible first
	_, err := p.page.WaitForSelector(action.Selector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000), // 10 second timeout
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for element %s: %w", action.Selector, err)
	}

	// Wait for element to be actionable (not covered by other elements)
	_, err = p.page.WaitForFunction(fmt.Sprintf(`
		() => {
			const element = document.querySelector('%s');
			return element && !element.disabled && element.offsetWidth > 0 && element.offsetHeight > 0;
		}
	`, action.Selector), playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(5000),
	})
	if err != nil {
		return nil, fmt.Errorf("element %s is not clickable: %w", action.Selector, err)
	}

	err = p.page.Click(action.Selector, playwright.PageClickOptions{
		Timeout: playwright.Float(5000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to click element %s: %w", action.Selector, err)
	}
	return "clicked", nil
}

func (p *PlaywrightAgent) handleType(action interfaces.BrowserAction) (interface{}, error) {
	// Wait for the element to be available and visible first
	_, err := p.page.WaitForSelector(action.Selector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000), // 10 second timeout
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for element %s: %w", action.Selector, err)
	}

	// Additional wait to ensure the element is fully interactive
	_, err = p.page.WaitForFunction(fmt.Sprintf(`
		() => {
			const element = document.querySelector('%s');
			return element && !element.disabled && element.offsetWidth > 0 && element.offsetHeight > 0;
		}
	`, action.Selector), playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(5000),
	})
	if err != nil {
		return nil, fmt.Errorf("element %s is not interactive: %w", action.Selector, err)
	}

	// Try clicking the element first to ensure it's focused
	err = p.page.Click(action.Selector, playwright.PageClickOptions{
		Timeout: playwright.Float(5000),
	})
	if err != nil {
		p.logger.WithField("error", err).Warn("Failed to click element before typing, continuing anyway")
	}

	// Clear the field first, then type the new value
	err = p.page.Fill(action.Selector, "")
	if err != nil {
		return nil, fmt.Errorf("failed to clear element %s: %w", action.Selector, err)
	}

	// Type the value with a small delay between characters for better reliability
	err = p.page.Type(action.Selector, action.Value, playwright.PageTypeOptions{
		Delay: playwright.Float(50), // 50ms delay between keystrokes
	})
	if err != nil {
		return nil, fmt.Errorf("failed to type in element %s: %w", action.Selector, err)
	}

	return "typed", nil
}

func (p *PlaywrightAgent) handleSelect(action interfaces.BrowserAction) (interface{}, error) {
	_, err := p.page.SelectOption(action.Selector, playwright.SelectOptionValues{
		Values: &[]string{action.Value},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to select option in %s: %w", action.Selector, err)
	}
	return "selected", nil
}

func (p *PlaywrightAgent) handleWait(action interfaces.BrowserAction) (interface{}, error) {
	selector, ok := action.Parameters["selector"].(string)
	if !ok {
		return nil, fmt.Errorf("selector parameter is required for wait action")
	}

	timeout := 5000.0 // default 5 seconds
	if timeoutParam, ok := action.Parameters["timeout"]; ok {
		if timeoutFloat, ok := timeoutParam.(float64); ok {
			timeout = timeoutFloat
		}
	}

	_, err := p.page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for selector %s: %w", selector, err)
	}
	return "waited", nil
}

func (p *PlaywrightAgent) handleScroll(action interfaces.BrowserAction) (interface{}, error) {
	pixels := 0
	if pixelsParam, ok := action.Parameters["pixels"]; ok {
		if pixelsFloat, ok := pixelsParam.(float64); ok {
			pixels = int(pixelsFloat)
		}
	}

	_, err := p.page.Evaluate(fmt.Sprintf("window.scrollBy(0, %d)", pixels))
	if err != nil {
		return nil, fmt.Errorf("failed to scroll: %w", err)
	}
	return "scrolled", nil
}

func (p *PlaywrightAgent) handleExtractText(action interfaces.BrowserAction) (interface{}, error) {
	text, err := p.page.TextContent(action.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from %s: %w", action.Selector, err)
	}
	return text, nil
}

func (p *PlaywrightAgent) handleExtractAttribute(action interfaces.BrowserAction) (interface{}, error) {
	attrName, ok := action.Parameters["attribute"].(string)
	if !ok {
		return nil, fmt.Errorf("attribute parameter is required for extract_attribute action")
	}

	attr, err := p.page.GetAttribute(action.Selector, attrName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract attribute %s from %s: %w", attrName, action.Selector, err)
	}
	return attr, nil
}
