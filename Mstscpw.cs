using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.ComponentModel;
using System.Security.Cryptography;
using System.Linq;
using System.Text;

namespace RDP_Portal {
    /**
     * By ChipForster
     * https://www.remkoweijnen.nl/blog/2007/10/18/how-rdp-passwords-are-encrypted/
     */
    class Mstscpw {
        private const int CRYPTPROTECT_UI_FORBIDDEN = 0x1;

        // Wrapper for the NULL handle or pointer.
        static private IntPtr NullPtr = ((IntPtr)((int)(0)));

        // Wrapper for DPAPI CryptProtectData function.
        [DllImport( "crypt32.dll", SetLastError = true,
        CharSet = System.Runtime.InteropServices.CharSet.Auto )]

        private static extern bool CryptProtectData(
            ref DATA_BLOB pPlainText,
            [MarshalAs(UnmanagedType.LPWStr)] string szDescription,
            IntPtr pEntroy,
            IntPtr pReserved,
            IntPtr pPrompt,
            int dwFlags,
            ref DATA_BLOB pCipherText);

        // BLOB structure used to pass data to DPAPI functions.
        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        internal struct DATA_BLOB {
            public int cbData;
            public IntPtr pbData;
        }

        private static void InitBLOB(byte[] data, ref DATA_BLOB blob) {
            blob.pbData = Marshal.AllocHGlobal(data.Length);
            if (blob.pbData == IntPtr.Zero)
                throw new Exception( "Unable to allocate buffer for BLOB data." );

            blob.cbData = data.Length;
            Marshal.Copy(data, 0, blob.pbData, data.Length);
        }

        public string encryptpw(string pw) {
            byte[] pwba = Encoding.Unicode.GetBytes(pw);
            DATA_BLOB dataIn = new DATA_BLOB();
            DATA_BLOB dataOut = new DATA_BLOB();
            StringBuilder epwsb = new StringBuilder();
            try {
                try {
                    InitBLOB(pwba, ref dataIn);
                } catch (Exception ex) {
                    throw new Exception( "Cannot initialize dataIn BLOB.", ex );
                }

                bool success = CryptProtectData(
                    ref dataIn, 
                    "psw",
                NullPtr,
                NullPtr,
                NullPtr,
                CRYPTPROTECT_UI_FORBIDDEN,
                    ref dataOut );

                if (!success) {
                    int errCode = Marshal.GetLastWin32Error();
                    throw new Exception( "CryptProtectData failed.", new Win32Exception(errCode) );
                }

                byte[] epwba = new byte[dataOut.cbData];
                Marshal.Copy(dataOut.pbData, epwba, 0, dataOut.cbData);

                // Convert hex data to hex characters (suitable for a string)
                for (int i = 0; i < dataOut.cbData; i++) {
                    epwsb.Append(Convert.ToString(epwba[i], 16).PadLeft(2, '0').ToUpper());
                }

            } catch (Exception ex) {
                throw new Exception( "unable to encrypt data.", ex );
            } finally {
                if (dataIn.pbData != IntPtr.Zero)
                    Marshal.FreeHGlobal(dataIn.pbData);

                if (dataOut.pbData != IntPtr.Zero)
                    Marshal.FreeHGlobal(dataOut.pbData);
            }

            return epwsb.ToString();
        }
    }
}
