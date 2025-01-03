package main

var (
	helpText = `
ktx - kubernetes context switcher

USAGE:

  ktx [OPTIONS|[config alias [context alias]]]

OPTIONS:

  [-]                 - Switch to the previous context

  [-h|-help]          - Shows this help.
  
  [-c|-is|-current]   - Returns the current context.

  [-v|-version]       - Prints the version

ALIASES:
  
  [config alias]      - Shows only the contexts of one kubeconfigs (given by its alias).
                        (DEFAULT: <no filter>) 

  [context alias]     - Directly switch to the context (given by its alias)
                        (DEFAULT: <no filter>) 


ENVIRONMENT VARIABLES:

  KTX_CONFIG_DIR      - Specify the directory for the "ktx" config files.
                        (DEFAULT: is OS related directory like "$HOME/.config/ktx" or "%AppData%\ktx")


FILES:

  config.jsonnet      - The global configuration file for all kubeconfigs and contexts. You can use this file 
                        to generate an alias per contexts or to add more descriptions for each context.

  contexts_<...>.yaml - The contexts files are in sync with the related kubeconfigs. You can directly edit
                        them in order to set for example the default namespace or add an alias.

  .libsonnet          - This is a helper file for the "config.jsonnet" and only imports all existing
                        "contexts_<...>.yaml" file. It also makes these contents available under the alias
                        of the related kubeconfig. This file doesn't need to be touched.

  .state              - The state file stores the last used kubeconfig and context together with the current
                        ones. This file is required for the "ktx -" command in order to jump back and forth
                        between two contexts.

EXAMPLES:

- Opens the TUI with all contexts of all kubeconfig files:
   
  ktx
  
- Opens the TUI with all contexts of a kubeconfig file with the alias "m":
   
  ktx m

- Opens the TUI with contexts filtered by "dev" of the kubeconfig with the alias "m":

  ktx m dev

- Switches to the previous context:

  ktx -

- Switches directly to the context with the name/alias "lab":

  ktx -c lab

- Returns the current context:

  ktx -c
`
)
