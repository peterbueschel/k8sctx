{
  // import all contexts_*.yaml files in the config folder
  _contexts_files:: import 'glob-str.stem://contexts_*.yaml',
  _aliases:: [k.alias for k in $.kube_configs],
  // parses the Yaml content of each contexts_*.yaml file and also make each
  // file available under the alias, like "m" from "contexts_m" where "m" is the alias
  contexts::
    if $._contexts_files == []
    then { [alias]: [] for alias in self._aliases }
    else {
      [std.split(name, '_')[1]]: std.parseYaml($._contexts_files[name])
      for name in std.objectFieldsAll($._contexts_files)
    },
}
