let pkgs = import <nixpkgs> { };
in pkgs.mkShell {
  # GOOS = "js";
  # GOARCH = "wasm";

  buildInputs = with pkgs; [
    libx11
    libxrandr
    libGL
    libGLX
    libxcursor
    libxinerama
    xinput
    libxi
    libXxf86vm
    libXext
  ];

  shellHook = ''
    EBITENGINE_LIBGL="${pkgs.libGL}/lib/libGL.so"; export EBITENGINE_LIBGL; EBITENGINE_LIBGLESv2="${pkgs.libGL}/lib/libGLESv2.so"; export EBITENGINE_LIBGLESv2; export LD_LIBRARY_PATH="${pkgs.libGL}/lib:$LD_LIBRARY_PATH";'';
}
