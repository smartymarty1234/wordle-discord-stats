use std::process::Command;

fn run(image: &str) -> String {
    let output = Command::new(env!("CARGO_BIN_EXE_imgparse"))
        .arg(format!("tests/{image}"))
        .output()
        .expect("failed to run imgparse");
    assert!(output.status.success(), "imgparse exited with error");
    String::from_utf8(output.stdout).unwrap().trim().to_string()
}

#[test]
fn single_users() {
    assert_eq!(
        run("single-users.png"),
        "wordle: 1752\ngrid 1: GBBGB GBGGB GGGGB GGGGG BBBBB BBBBB"
    );
}

#[test]
fn three_users() {
    assert_eq!(
        run("three-users.png"),
        "wordle: 1751\n\
         grid 1: BBBYB BGYBY GGGGG BBBBB BBBBB BBBBB\n\
         grid 2: BYBBY YBBYY GGGGG BBBBB BBBBB BBBBB\n\
         grid 3: BBBBY BYBYB BYYGB GGBGB GGGGG BBBBB"
    );
}
