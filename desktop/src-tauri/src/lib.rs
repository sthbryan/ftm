pub mod commands;
use log::{error, info};
use std::env;
use std::io::{BufRead, BufReader};
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::thread;
use std::time::Duration;

const PORT_RANGE_START: u16 = 40510;
const PORT_RANGE_END: u16 = 40550;
const WEB_PORT_ENV: &str = "FOUNDRY_TUNNEL_WEB_PORT";

fn find_available_port() -> u16 {
    for port in PORT_RANGE_START..=PORT_RANGE_END {
        if std::net::TcpListener::bind(format!("0.0.0.0:{}", port)).is_ok() {
            return port;
        }
    }
    panic!(
        "No available port found in range {}-{}",
        PORT_RANGE_START, PORT_RANGE_END
    );
}

pub fn find_or_build_binary() -> PathBuf {
    let project_root = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .parent()
        .unwrap()
        .to_path_buf();

    let binary_path = project_root.join("bin").join("ftm");

    if binary_path.exists() {
        info!("Found ftm binary at {:?}", binary_path);
        return binary_path;
    }

    let go_module = project_root.join("cmd").join("ftm");
    if !go_module.exists() {
        error!("Cannot find Go module at {:?}", go_module);
        panic!("Go module not found: {:?}", go_module);
    }

    let bin_dir = project_root.join("bin");
    let _ = std::fs::create_dir_all(&bin_dir);

    info!("Building ftm binary...");
    let output = Command::new("go")
        .args(["build", "-o", binary_path.to_str().unwrap(), "./cmd/ftm"])
        .current_dir(&project_root)
        .output()
        .expect("Failed to build ftm binary");

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        error!("Go build failed: {}", stderr);
        panic!("Failed to build ftm binary");
    }

    info!("ftm binary built successfully");
    binary_path
}

pub fn start_ftm_server(binary_path: &PathBuf) -> u16 {
    let port = find_available_port();

    let project_root = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .parent()
        .unwrap()
        .to_path_buf();

    let mut child = Command::new(binary_path)
        .arg("--server")
        .arg("--port")
        .arg(port.to_string())
        .current_dir(&project_root)
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()
        .expect("Failed to start ftm server");

    if let Some(stdout) = child.stdout.take() {
        let reader = BufReader::new(stdout);
        thread::spawn(move || {
            for line in reader.lines() {
                if let Ok(line) = line {
                    info!("[ftm] {}", line);
                }
            }
        });
    }

    wait_for_server(port);
    port
}

fn wait_for_server(port: u16) {
    let max_attempts = 30;
    let url = format!("http://localhost:{}/api/status", port);

    for attempt in 1..=max_attempts {
        match ureq::get(&url).call() {
            Ok(_) => {
                info!("Server is ready on port {}", port);
                return;
            }
            Err(_) => {
                if attempt == max_attempts {
                    error!("Server failed to start after {} attempts", max_attempts);
                    panic!("Server failed to start");
                }
                thread::sleep(Duration::from_millis(500));
            }
        }
    }
}

pub fn setup_app(_app: &tauri::App) -> Result<u16, Box<dyn std::error::Error>> {
    let port = start_ftm_server(&find_or_build_binary());
    env::set_var(WEB_PORT_ENV, port.to_string());
    info!("FTM server running on port {}", port);
    Ok(port)
}
