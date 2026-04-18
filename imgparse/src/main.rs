use ocrs::{ImageSource, OcrEngine, OcrEngineParams};
use rten::Model;
use std::io::Read;
use std::path::PathBuf;
use std::time::Instant;

macro_rules! timed {
    ($debug:expr, $label:expr, $block:expr) => {{
        let t = Instant::now();
        let result = $block;
        if $debug {
            eprintln!("[debug] {}: {:?}", $label, t.elapsed());
        }
        result
    }};
}

fn load_engine(debug: bool) -> Option<OcrEngine> {
    let models_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("models");
    let detection_model = timed!(debug, "load detection model",
        Model::load_file(models_dir.join("text-detection.rten")).ok()?);
    let recognition_model = timed!(debug, "load recognition model",
        Model::load_file(models_dir.join("text-recognition.rten")).ok()?);
    timed!(debug, "init engine", OcrEngine::new(OcrEngineParams {
        detection_model: Some(detection_model),
        recognition_model: Some(recognition_model),
        ..Default::default()
    })
    .ok())
}

fn extract_wordle_number(engine: &OcrEngine, img: &image::DynamicImage, debug: bool) -> Option<u32> {
    let rgb = timed!(debug, "to_rgb8", img.to_rgb8());
    let img_source = ImageSource::from_bytes(rgb.as_raw(), rgb.dimensions()).ok()?;
    let ocr_input = timed!(debug, "prepare_input", engine.prepare_input(img_source).ok()?);
    let text = timed!(debug, "get_text", engine.get_text(&ocr_input).ok()?);

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

fn load_image(src: &str, debug: bool) -> image::DynamicImage {
    if src.starts_with("http://") || src.starts_with("https://") {
        timed!(debug, "fetch image", {
            let resp = ureq::get(src).call().expect("failed to fetch image");
            let mut bytes = Vec::new();
            resp.into_reader().read_to_end(&mut bytes).expect("failed to read image bytes");
            image::load_from_memory(&bytes).expect("failed to decode image")
        })
    } else {
        timed!(debug, "open image", image::open(src).expect("failed to open image"))
    }
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let debug = args.iter().any(|a| a == "--debug");
    let src = args.iter().skip(1).find(|a| *a != "--debug")
        .expect("usage: imgparse [--debug] <url-or-path>");
    let engine = load_engine(debug).expect("failed to load OCR engine");
    let img = load_image(src, debug);
    match extract_wordle_number(&engine, &img, debug) {
        Some(n) => println!("{}", n),
        None => {
            eprintln!("could not extract wordle number");
            std::process::exit(1);
        }
    }
}
