use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int};

use json::JsonValue;
use lazy_static::lazy_static;
use std::sync::{Mutex, RwLock};

use smcore::{smh, smu};
use smdton::{SmDton, SmDtonBuffer, SmDtonBuilder, SmDtonMap};

type GoCallNative = extern "C" fn(*const c_char) -> *mut c_char;


lazy_static! {
    static ref LIB_DATA: RwLock<LibData> = RwLock::new(LibData { _sn: 0 });
    static ref CALLBACK: Mutex<Option<GoCallNative>> = Mutex::new(None);
}

struct LibData {
    _sn: u64,
}

fn inc_sn() -> u64 {
    let mut save_env = false;
    let sn: u64;
    {
        let mut hb = LIB_DATA.write().unwrap();
        if hb._sn == 0 {
            save_env = true;
        }
        hb._sn += 1;
        sn = hb._sn;
    }

    if save_env {
        smloadwasm::init();
        smsys::init();
    }

    return sn;
}

#[no_mangle]
pub extern "C" fn smwasm_load(_wasm: *const c_char, _space: c_int) {
    let c_str = unsafe { CStr::from_ptr(_wasm) };
    match c_str.to_str() {
        Ok(itxt) => {
            smloadwasm::load_wasm(itxt, _space);
        },
        Err(_) => {
        }
    }
}

#[no_mangle]
pub extern "C" fn smwasm_sn() -> c_int {
    let ret = inc_sn() as i32;
    return ret;
}

#[no_mangle]
pub extern "C" fn smwasm_call(_intxt: *const c_char) -> *mut c_char {
    let mut otxt = "{}".to_string();

    let c_str = unsafe { CStr::from_ptr(_intxt) };
    match c_str.to_str() {
        Ok(itxt) => {
            let jsn = json::parse(&itxt).unwrap();
            let smb = smu.build_buffer(&jsn);
            let ret = smh.call(smb);
        
            let op_ret = ret.stringify();
            match op_ret {
                Some(txt) => {
                    otxt = txt;
                },
                None => {
                },
            }
        },
        Err(_) => {
        }
    }

    let result = CString::new(otxt).unwrap();
    return result.into_raw();
}

#[no_mangle]
pub extern "C" fn smwasm_register(_usage: *const c_char) -> c_int {
    if _usage.is_null() {
        return 0;
    }

    let c_str = unsafe { CStr::from_ptr(_usage) };
    match c_str.to_str() {
        Ok(input) => {
            let _define = json::parse(&input).unwrap();
            smh.register_by_json(&_define, _call_sm);
            return 1;
        },
        Err(_) => {
        }
    }
    0
}

#[no_mangle]
pub extern "C" fn smwasm_set_above(_callback: GoCallNative) {
    inc_sn();

    let mut cb = CALLBACK.lock().unwrap();
    *cb = Some(_callback);
}

fn call_native(_intxt: &str) -> String {
    let mut otxt = "{}".to_string();

    let cb = CALLBACK.lock().unwrap();
    if let Some(callback) = *cb {
        let c_string = CString::new(_intxt).unwrap();
        let _inptr = c_string.as_ptr();
        let c_output = callback(_inptr);

        let outtxt = unsafe {
            CStr::from_ptr(c_output)
                .to_str()
                .expect("Failed to convert CStr to &str")
                .to_string()
        };

        otxt = outtxt;

        unsafe {
            libc::free(c_output as *mut libc::c_void);
        }
    } else {
    }

    return otxt;
}

fn _call_sm(_input: &SmDtonBuffer) -> SmDtonBuffer {
    let sd = SmDton::new_from_buffer(_input);
    let intxt = sd.stringify().unwrap();

    let result_str = call_native(&intxt);
    let parsed: Result<JsonValue, json::Error> = json::parse(&result_str);
    match parsed {
        Ok(jsn) => {
            let mut sdb = SmDtonBuilder::new_from_json(&jsn);
            return sdb.build();
        }
        Err(_e) => {
        }
    }

    let mut _map = SmDtonMap::new();
    return _map.build();
}
