{
  description = "mts - modern timestamp filter";

  inputs = { nixpkgs.url = "nixpkgs/nixos-unstable"; };
  outputs = { self, nixpkgs }:
    let
      version = "dev";
      expectedGoVersion = "1.26";
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs {
        inherit system;
        overlays = [
          (final: prev: {
            go = final.go_1_26;
          })
        ];
      });
    in {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
          buildDate = pkgs.runCommand "build-date" {} ''
            date -u +%Y-%m-%dT%H:%M:%SZ > $out
          '';
          buildDateStr = builtins.readFile buildDate;
          buildCommit =
            if self ? dirtyShortRev && self.dirtyShortRev != null then self.dirtyShortRev
            else if self ? shortRev && self.shortRev != null then self.shortRev
            else "unknown";
        in assert nixpkgs.lib.assertMsg
          (nixpkgs.lib.versions.majorMinor pkgs.go.version == expectedGoVersion)
          "Go toolchain mismatch: nixpkgs provides ${pkgs.go.version}, expected ${expectedGoVersion}.x";
        {
          mts = pkgs.buildGoModule {
            pname = "mts";
            version = "${version}";
            src = self;
            modRoot = "src";

            go = pkgs.go;

            # Separate man output: docs install only when requested
            # (nix build .#mts^man, or downstream docs enabled).
            # Default installs stay binary-only via outputsToInstall.
            outputs = [ "out" "man" ];

            meta = {
              description = "Modern timestamp filter: prepend timestamps to log lines, wrap commands with exit-code passthrough.";
              homepage = "https://github.com/nfrastack/mts";
              maintainer = [ "nfrastck" "tiredofit" ];
              license = "BSD-3-Clause";
              mainProgram = "mts";
              outputsToInstall = [ "out" ];
            };

            ldflags = [
              "-s"
              "-w"
              "-X main.Version=${version}"
              "-X main.buildCommit=${buildCommit}"
              "-X main.BuildTime=${buildDateStr}"
            ];

            vendorHash = null;

            postInstall = ''
              mkdir -p $man/share/man/man1
              cp ${./man/mts.1} $man/share/man/man1/mts.1
            '';
          };
          default = self.packages.${system}.mts;
        });

      devShells = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              gnumake
              gnused
              go
            ];
          };
        });

      devShell = forAllSystems (system: self.devShells.${system}.default);
    };
}
