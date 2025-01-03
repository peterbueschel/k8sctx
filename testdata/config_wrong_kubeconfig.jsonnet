local split(name) = {
  local parts = std.split(name, ':'),
  name: name,
  environment: parts[1],
  region: parts[3],
  cluster: parts[4],
};

{
  kube_configs: [{
    alias: 't',
    path: 'testdata/does not exists',
    context_file: 'contexts.yaml',
    contexts: std.map(function(c) c + split(c.name), std.parseYaml(importstr 'contexts.yaml')),
  }],
}
