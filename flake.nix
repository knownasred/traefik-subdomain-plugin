{
  description = "Development environment for the Traefik Yaegi plugin";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";

  outputs = { self, nixpkgs }:
    let
      # nixpkgs marks Yaegi as broken on Darwin.
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      # Yaegi resolves this plugin's test imports through GOPATH, not go.mod.
      setupYaegi = ''
        export GOPATH="$PWD/.nix-go"
        modulePath=$(go list -m)
        mkdir -p "$GOPATH/src/$(dirname "$modulePath")"
        ln -sfn "$PWD" "$GOPATH/src/$modulePath"
        unset modulePath
      '';
    in
    {
      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            packages = with pkgs; [ go yaegi gopls gnumake ];
            GOTOOLCHAIN = "local";
            CGO_ENABLED = "0";
            shellHook = setupYaegi;
          };
        });

      checks = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          tests = pkgs.runCommand "plugin-tests" {
            nativeBuildInputs = with pkgs; [ go yaegi gnumake ];
            GOTOOLCHAIN = "local";
            CGO_ENABLED = "0";
          } ''
            export HOME="$TMPDIR"
            export GOCACHE="$TMPDIR/go-cache"
            cp -r ${self} source
            chmod -R u+w source
            cd source
            ${setupYaegi}
            make test
            make yaegi_test
            touch "$out"
          '';
        });
    };
}
