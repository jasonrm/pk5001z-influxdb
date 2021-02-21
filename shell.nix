{
  pkgs ? import <nixpkgs> { },
  mkShell ? pkgs.mkShell,
}:
mkShell rec {
  buildInputs = with pkgs; [
    go
  ];

  shellHook = ''
    if [ -f secrets.env ]; then
        set -o allexport
        source secrets.env
        set +o allexport
    fi
  '';
}
