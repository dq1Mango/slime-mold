let pkgs = import <nixpkgs> { };
in pkgs.mkShell {
  GOOS = "js";
  GOARCH = "wasm";
}
