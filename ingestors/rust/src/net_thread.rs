use crate::{
    process,
    trace::{ApiTrace, ProcessTraceRes},
};
use lazy_static::lazy_static;
use std::{
    panic::{catch_unwind, AssertUnwindSafe},
    thread,
};
use tokio::{
    runtime,
    sync::mpsc::{channel, Receiver, Sender},
};

lazy_static! {

/// The channel that traces get sent to.
pub static ref SEND_CHANNEL: Sender<(ApiTrace, Option<ProcessTraceRes>)> = {
    // TODO: The size should probably be configurable, whether by the user or not.
    let (send, recv) = channel(100);
    thread::spawn(|| main(recv));
    send
};

}

/// The main function for the network thread.
fn main(mut recv: Receiver<(ApiTrace, Option<ProcessTraceRes>)>) -> ! {
    // The async runtime.
    let runtime = runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .expect("Failed to create tokio runtime");

    loop {
        // TODO: verify that this is fine
        let result = catch_unwind(AssertUnwindSafe(|| -> ! {
            runtime.block_on(async {
                loop {
                    main_loop(&mut recv).await
                }
            })
        }));

        if let Err(err) = result {
            // TODO: Log the panic.
        }
    }
}

/// The inner loop of the network thread.
async fn main_loop(recv: &mut Receiver<(ApiTrace, Option<ProcessTraceRes>)>) {
    let (trace, process_results) = recv
        .recv()
        .await
        .expect("Somehow the SEND_CHANNEL got dropped?");

    let process_results = process_results.unwrap_or_else(|| process(&trace));

    // TODO
}
