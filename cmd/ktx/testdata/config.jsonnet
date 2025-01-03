(import '.libsonnet') +
{
  //  run "ktx" to open all kube configs
  kube_configs: [
    {
      alias: 't',  // run "ktx {{ $kubeConfig.Alias }}" to open only this config
      path: 'testdata/kube.config',
      contexts: std.mapWithIndex(
        function(i, c) { index: '' + i } + c, std.get($.contexts, self.alias, [])
      ),
    },
  ],
}
