import os
import json

BOOST_DIR = 'D:\\Development\\Code\\C++\\boost'
EXT_DIR = '//third_party/boost/%s'
SRC_EXTS = ['c', 'cpp', 'cc']

MODULE_DEPS = dict(
  filesystem=['config', 'system', 'type_traits', 'detail', 'iterator', 'smart_ptr', 'io', 'functional', 'range'],
  system=['config', 'predef', 'assert', 'core'],
  iterator=['mpl', 'static_assert'],
  mpl=['preprocessor'],
  smart_ptr=['throw_exception'],
  range=['concept_check', 'utility'],
  algorithm=['function'],
)

# MakeBaseConfig makes the basic configuration file for a module and returns it.
def MakeBaseExternalConfig(module):
  return dict(
    url="https://github.com/boostorg/%s" % module,
    branch="boost-1.62.0",
  )

# MakeBuildConfig makes the basic build configuration for a module.
def MakeBaseBuildConfig(module):
  return {
    module: dict(
      type="c++/library",
      hdrs=["glob:include/**/*.hpp", "glob:include/**/*.h"],
      includes=["include"],

      # Platform specific requirements.
      linux=dict(
        compile_flags=["-std=c++11"],
      ),

      windows=dict(
        compile_flags=["-DBOOST_ALL_NO_LIB"],
      ),
    )
  }

# Get a list of all modules.
modules = [m for m in os.listdir(os.path.join(BOOST_DIR, 'libs')) if '.' not in m]

# For each module...
finalCfg = dict(external={})
for module in modules:
  moduleRoot = os.path.join(BOOST_DIR, 'libs', module)

  # Make the base BUILD config.
  buildCfg = MakeBaseBuildConfig(module)

  # Check for src files.
  srcDirPath = os.path.join(moduleRoot, 'src')
  srcFiles = []
  windowsSrcFiles = []
  linuxSrcFiles = []
  if os.path.exists(srcDirPath) and os.path.isdir(srcDirPath):
    for base, _, files in os.walk(srcDirPath):
      for file in files:
        srcWorkspacePath = os.path.relpath(os.path.join(base, file), moduleRoot)
        srcWorkspacePath = srcWorkspacePath.replace(os.sep, '/')
        if file.rsplit('.', 1)[-1] in SRC_EXTS:
          if 'windows' in srcWorkspacePath:
            windowsSrcFiles.append(srcWorkspacePath)
          elif 'linux' in srcWorkspacePath or 'posix' in srcWorkspacePath:
            linuxSrcFiles.append(srcWorkspacePath)
          else:
            srcFiles.append(srcWorkspacePath)

  if srcFiles:
    buildCfg[module]['srcs'] = sorted(srcFiles)

  if windowsSrcFiles:
    buildCfg[module]['windows']['srcs'] = sorted(windowsSrcFiles)

  if linuxSrcFiles:
    buildCfg[module]['linux']['srcs'] = sorted(linuxSrcFiles)

  # Add dependencies.
  deps = MODULE_DEPS.get(module, [])
  if deps:
    buildCfg[module]['deps'] = [EXT_DIR % d for d in sorted(deps)]

  # Build the external config.
  externalCfg = MakeBaseExternalConfig(module)
  externalCfg['build'] = buildCfg
  finalCfg['external'][EXT_DIR % module] = externalCfg

# Print the final config as json.
print(json.dumps(finalCfg, indent=4, sort_keys=True))
