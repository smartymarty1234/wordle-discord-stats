use ocrs::{ImageSource, OcrEngine, OcrEngineParams};
use rten::Model;
use std::io::Read;
use std::path::PathBuf;

fn extract_wordle_number(img: &image::DynamicImage) -> Option<u32> {
    let models_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("models");
    let detection_model = Model::load_file(models_dir.join("text-detection.rten")).ok()?;
    let recognition_model = Model::load_file(models_dir.join("text-recognition.rten")).ok()?;
    let engine = OcrEngine::new(OcrEngineParams {
        detection_model: Some(detection_model),
        recognition_model: Some(recognition_model),
        ..Default::default()
    })
    .ok()?;

    let rgb = img.to_rgb8();
    let img_source = ImageSource::from_bytes(rgb.as_raw(), rgb.dimensions()).ok()?;
    let ocr_input = engine.prepare_input(img_source).ok()?;
    let text = engine.get_text(&ocr_input).ok()?;

    let words: Vec<&str> = text.split_whitespace().collect();
    for i in 0..words.len().saturating_sub(1) {
        if words[i].eq_ignore_ascii_case("no.") || words[i].eq_ignore_ascii_case("no") {
            if let Ok(n) = words[i + 1].parse::<u32>() {
                return Some(n);
            }
        }
    }
    None
}

fn load_image(src: &str) -> image::DynamicImage {
    if src.starts_with("http://") || src.starts_with("https://") {
        let resp = ureq::get(src).call().expect("failed to fetch image");
        let mut bytes = Vec::new();
        resp.into_reader().read_to_end(&mut bytes).expect("failed to read image bytes");
        image::load_from_memory(&bytes).expect("failed to decode image")
    } else {
        image::open(src).expect("failed to open image")
    }
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let src = args.iter().find(|a| *a != &args[0])
        .expect("usage: imgparse <url-or-path>");
    let img = load_image(src);
    match extract_wordle_number(&img) {
        Some(n) => println!("{}", n),
        None => {
            eprintln!("could not extract wordle number");
            std::process::exit(1);
        }
    }
}
