package config

import "github.com/fatih/color"

var Banner = `
██╗  ██╗███████╗██████╗ ██████╗ ███████╗ ██████╗███████╗
██║ ██╔╝██╔════╝██╔══██╗██╔══██╗██╔════╝██╔════╝██╔════╝
█████╔╝ █████╗  ██████╔╝██████╔╝█████╗  ██║     ███████╗
██╔═██╗ ██╔══╝  ██╔══██╗██╔══██╗██╔══╝  ██║     ╚════██║
██║  ██╗███████╗██║  ██║██████╔╝███████╗╚██████╗███████║
╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═════╝ ╚══════╝ ╚═════╝╚══════╝
`

func PrintStartupBanner(env string) {
	banner := color.New(color.Bold, color.FgHiBlue).PrintlnFunc()
	banner(Banner)
	version := color.New(color.Bold, color.FgBlue).PrintlnFunc()
	version("Running v" + Version + " [ENV: " + env + "]")
	println()
}
