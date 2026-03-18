use std::fs;
use std::path::Path;
use std::process::Command;

fn main() {
    let go_binary_name = if cfg!(target_os = "windows") {
        "ftm.exe"
    } else {
        "ftm"
    };

    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").unwrap();
    let bin_dir = Path::new(&manifest_dir).join("bin");
    fs::create_dir_all(&bin_dir).ok();
    let target_path = bin_dir.join(go_binary_name);

    println!("cargo::warning=Building Go server to {:?}", target_path);

    let output = Command::new("go")
        .args(&["build", "-o", target_path.to_str().unwrap(), "./cmd/ftm"])
        .current_dir(format!("{}/../..", manifest_dir))
        .output()
        .expect("Failed to compile Go server");

    if !output.status.success() {
        eprintln!(
            "Go build failed: {}",
            String::from_utf8_lossy(&output.stderr)
        );
        panic!("Go server compilation failed");
    }

    println!("cargo::warning=Go server compiled successfully");
    tauri_build::build();
}
