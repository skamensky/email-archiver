import ReactModal from 'react-modal';
import {ReactNode} from 'react';
import {buttonClass} from "./utils";

export const Modal=({open,closeReq,title,content}:{open:boolean,closeReq:()=>void,title:string,content:ReactNode})=>{

    const customStyles = {
        content: {
            top: '50%',
            left: '50%',
            right: 'auto',
            bottom: 'auto',
            marginRight: '-50%',
            transform: 'translate(-50%, -50%)',
            maxWidth: '80vw', // Optional: restrict width
            width: 'auto', // Auto width based on content
            borderRadius: '8px', // Optional: for rounded corners
            maxHeight: '80vh', // Optional: takes scaling factor into account, making it responsive
        },
        overlay: {
            backgroundColor: 'rgba(0, 0, 0, 0.75)' // Optional: for overlay customization
        }
    };



    return <ReactModal
        isOpen={open}
        contentLabel={title}
        shouldCloseOnOverlayClick={true}
        onRequestClose={() => {
            closeReq();
        }}
        preventScroll={true}
        style={customStyles}
    >
        <div className="flex flex-col h-full">
            <div className="bg-gray-100 p-4 rounded-t sticky top-0">
                <h2 className="text-lg font-bold text-center">{title}</h2>
            </div>
            <div className="flex-grow overflow-auto p-4">
                {content}
            </div>
            {/*sticky footer: version of:

                <div className="p-4 bg-white">
                    <button className="w-full p-2 bg-gray-500 text-white rounded" onClick={closeReq}>
                    Close
                    </button>
                </div>
            */}
            <div className="sticky bottom-0">
                <button className={buttonClass('gray')} onClick={closeReq}>
                    Close
                </button>
            </div>
        </div>
    </ReactModal>


}
