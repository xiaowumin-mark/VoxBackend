const ReactDOMGlobal = ReactDOM;

const callImmediately = (callback) => callback();

export default ReactDOMGlobal;
export const createPortal = ReactDOMGlobal.createPortal;
export const flushSync = ReactDOMGlobal.flushSync ?? callImmediately;
export const unstable_batchedUpdates = ReactDOMGlobal.unstable_batchedUpdates ?? callImmediately;
