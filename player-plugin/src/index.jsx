import Setting from './setting';
import MainContext from './context';
import LoadCssHoudini from './smooth-corners';
LoadCssHoudini();
extensionContext.addEventListener('extension-load', function () {
    console.log('load');
    
    extensionContext.registerPlayerSource("voxbackend");
    extensionContext.registerComponent("settings", Setting)
    extensionContext.registerComponent("context", MainContext);
});
extensionContext.addEventListener('extension-unload', function () {
    console.log('unload');
});

