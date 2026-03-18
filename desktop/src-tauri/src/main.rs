#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::Manager;
use std::process::Command;
use std::time::Duration;
use std::thread;
use std::net::TcpStream;

#[tokio::main]
async fn main() {
    tauri::Builder::default()
        .setup(|app| {
            let app_handle = app.handle().clone();
            
            std::thread::spawn(move || {
                if let Err(e) = start_server(&app_handle) {
                    eprintln!("Failed to start server: {}", e);
                }
            });

            #[cfg(debug_assertions)]
            if let Some(main_window) = app.get_webview_window("main") {
                main_window.open_devtools();
            }

            Ok(())
        })
        .on_window_event(|_window, event| {
            if let tauri::WindowEvent::CloseRequested { .. } = event {
                std::process::exit(0);
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn start_server(app_handle: &tauri::AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let exe_dir = std::env::current_exe()?
        .parent()
        .ok_or("Failed to get exe directory")?
        .to_path_buf();
    
    let server_exe = if cfg!(target_os = "windows") {
        exe_dir.join("ftm.exe")
    } else {
        exe_dir.join("ftm")
    };

    if !server_exe.exists() {
        return Err(format!("Server binary not found at {:?}", server_exe).into());
    }

    let _cmd = Command::new(&server_exe)
        .arg("-web")
        .arg("-port")
        .arg("7777")
        .spawn()?;

    for _ in 0..30 {
        if TcpStream::connect("127.0.0.1:7777").is_ok() {
            println!("Server ready at http://localhost:7777");
            thread::sleep(Duration::from_millis(500));
            
            let main_window = app_handle.get_webview_window("main")
                .ok_or("Failed to get main window")?;
            main_window.navigate(tauri::Url::parse("http://localhost:7777")?)?;
            
            return Ok(());
        }
        thread::sleep(Duration::from_millis(100));
    }

    Err("Server failed to start within 3 seconds".into())
}
