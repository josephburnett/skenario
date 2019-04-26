package serve

import (
	"testing"

	"github.com/sclevine/agouti"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAcceptance(t *testing.T, describe spec.G, it spec.S) {
	describe("the web GUI", func() {
		var driver *agouti.WebDriver
		var page *agouti.Page
		var err error

		it.Before(func() {
			driver = agouti.ChromeDriver(agouti.ChromeOptions("args", []string{
				"--headless",
				"--allow-insecure-localhost",
				"--no-sandbox",
			}), agouti.Debug)

			err = driver.Start()
			require.NoError(t, err)

			page, err = driver.NewPage()
			require.NoError(t, err)

			err = page.Navigate("http://localhost:3000?inmemory=true")
			assert.NoError(t, err)
		})

		it("is called Skenario", func() {
			title, err := page.Title()
			assert.NoError(t, err)
			assert.Equal(t, "Skenario", title)
		})

		describe("executing simulations", func() {
			it.Before(func() {
				setParams(t, page)

				btn := page.FindByButton("Execute simulation")
				require.NotNil(t, btn)

				err = btn.Click()
				require.NoError(t, err)
			})

			it("replaces the #loading <p> with a chart", func() {
				loading := page.FindByID("loading")
				assert.NotNil(t, loading)

				vegaEmbed := page.FindByClass("vega-embed")
				assert.NotNil(t, vegaEmbed)
			})
		})

		it.After(func() {
			err = driver.Stop()
			assert.NoError(t, err)
		})
	})
}

func setParams(t *testing.T, page *agouti.Page) {
	var err error

	var settings = map[string]string{
		"runFor":                      "10",
		"launchDelay":                 "5",
		"terminateDelay":              "1",
		"tickInterval":                "2",
		"stableWindow":                "60",
		"panicWindow":                 "6",
		"scaleToZeroGracePeriod":      "30",
		"targetConcurrencyDefault":    "1",
		"targetConcurrencyPercentage": "0.5",
		"maxScaleUpRate":              "100",

		"rampConfigMaxRPS": "10",
		"rampConfigDeltaV": "1",
	}

	for name, value := range settings {
		field := page.FindByID(name)
		require.NotNil(t, field)
		err = field.Fill(value)
		require.NoError(t, err)
	}

}
