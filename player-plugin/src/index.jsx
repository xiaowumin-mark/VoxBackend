import Setting from './setting';
import MainContext from './context';
import LoadCssHoudini from './smooth-corners';
import { ConnectBackend } from './ws';
LoadCssHoudini();
extensionContext.addEventListener('extension-load', function () {
    console.log('load');
    ConnectBackend();
    // 创建一个style 到meta
    const style = document.createElement('style');
    style.textContent = `
    @keyframes shine-move {
         0% {
            mask-position: 100% 0;
            -webkit-mask-position: 100% 0;
        }
        100% {
            mask-position: -100% 0;
            -webkit-mask-position: -100% 0;
        }`
    document.head.appendChild(style);

    

    extensionContext.registerComponent("settings", Setting)
    extensionContext.registerComponent("context", MainContext);
});
extensionContext.addEventListener('extension-unload', function () {
    console.log('unload');
});

