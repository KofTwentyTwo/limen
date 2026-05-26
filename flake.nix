{
  description = "limen — terminal launcher TUI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "limen";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-zYQ6hMFBFDPNlViB7EmnxbXozardyYeHiUSiG4uS/9Y=";

          ldflags = [
            "-s"
            "-w"
          ];

          meta = {
            description = "Terminal launcher TUI for tmux + ssh";
            homepage = "https://github.com/KofTwentyTwo/limen";
            license = pkgs.lib.licenses.gpl3Only;
            mainProgram = "limen";
          };
        };
      });
}
