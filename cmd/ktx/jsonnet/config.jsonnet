(import '.libsonnet') +
{
  //  run "ktx" to open all kube configs
  kube_configs: [
    {{- range $idx, $kubeConfig := . }}
    {
      alias: '{{ $kubeConfig.Alias }}', // run "ktx {{ $kubeConfig.Alias }}" to open only this config
      path: '{{ $kubeConfig.Path }}',
      contexts: std.map(
        function(c) { namespace: 'default', alias: c.name } + c, std.get($.contexts, self.alias, [])
      ),
    },
    {{- end }}
  ],
}
