<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<style type="text/css">
    .diskInfoClass {
        display: inline-block;
        padding: 10px;
        width: 150px;
        line-height: 20px;
    }
</style>
<div class="easyui-layout" data-options="fit:true,border:false">
    <div region="north" data-options="border:false" style="height: 300px;border-width: 0 0 1px 0">
        <table class="easyui-datagrid" title="CPU" fit="true" fitColumns="true"
               data-options="border:false,
                       singleSelect:true,onLoadSuccess:datagridLoadSuccess,
                       url: '${ctx}/system/resourceMonitor/getCPUInfo.action',
                       striped : true">
            <thead>
            <tr>
                <th data-options="field:'cpuMHZ'" width="30">MHZ</th>
                <th data-options="field:'cpuVendor'" width="50">Vendor</th>
                <th data-options="field:'cpuModel'" width="200">Model</th>
                <th data-options="field:'cpuWaitPt'" width="80">WaitPercent</th>
                <th data-options="field:'cpuErroPt'" width="80">ErrorPercent</th>
                <th data-options="field:'cpuUserUsage'" width="80">UserUsage</th>
                <th data-options="field:'cpuSystemUsage'" width="80">SystemUsage</th>
                <th data-options="field:'cpuTotalUsagePt'" width="80">TotalUsagePercent</th>
                <th data-options="field:'cpuFreePercent'" width="80">FreePercent</th>
            </tr>
            </thead>
        </table>
    </div>
    <div region="center" data-options="border:false">
        <div class="easyui-layout" data-options="fit:true,border:false">
            <div region="west" data-options="border:false"
                 style="width: 700px;border-width: 0px 1px 0px 0">
                <div class="easyui-layout" data-options="fit:true,border:false">
                    <div region="north" style="height: 100px;border-width: 0px 0px 1px 0px" data-options="border:false,collapsible:false" title="Current Database">
                        <table class="easyui-datagrid" fit="true" fitColumns="true"
                               data-options="border:false,
                               singleSelect:true,
                               url: '${ctx}/system/resourceMonitor/getDatabaseInfo.action',
                               striped : true,onLoadSuccess:datagridLoadSuccess">
                            <thead>
                            <tr>
                                <th data-options="field:'threads_connected'" width="100">ConnectedThreads</th>
                                <th data-options="field:'threads_running'" width="80">RunningThreads</th>
                                <th data-options="field:'max_connections'" width="80">MaxConnections</th>
                                <th data-options="field:'data_size'" width="80">DataSize(MB)</th>
                                <th data-options="field:'index_size'" width="80">IndexSize(MB)</th>
                                <th data-options="field:'free_size'" width="80">FreeSize(MB)</th>
                            </tr>
                            </thead>
                        </table>
                    </div>
                    <div region="center" title="History Database" data-options="border:false">
                        <table class="easyui-datagrid" fit="true" fitColumns="true"
                               data-options="border:false,
                                           singleSelect:true,
                                           idField: 'time',
                                           pagination: true,
                                           url: '${ctx}/system/resourceMonitor/getDBSpaceRecord.action',
                                           striped : true,onLoadSuccess:datagridLoadSuccess">
                            <thead>
                            <tr>
                                <th data-options="field:'time'" width="80">Day</th>
                                <th data-options="field:'data_size'" width="80">DataSize(MB)</th>
                                <th data-options="field:'index_size'" width="80">IndexSize(MB)</th>
                                <th data-options="field:'free_size'" width="80">FreeSize(MB)</th>
                                <th data-options="field:'increase_size'" width="80">IncreaseSize(MB)</th>
                            </tr>
                            </thead>
                        </table>
                    </div>
                </div>
            </div>
            <div region="center" data-options="border:false">
                <div class="easyui-layout" data-options="fit:true,border:false">
                    <div region="north" data-options="collapsible:false,border:false"
                         style="height: 100px;border-width: 0 0 1px 0;"  title="Memory">
                        <table class="easyui-datagrid" fit="true" fitColumns="true"
                               data-options="border:false,
                               singleSelect:true,
                               url: '${ctx}/system/resourceMonitor/getMemoryInfo.action',
                               striped : true,onLoadSuccess:datagridLoadSuccess">
                            <thead>
                            <tr>
                                <th data-options="field:'memoryTotal'" width="80">TotalMemory(MB)</th>
                                <th data-options="field:'memoryFree'" width="80">FreeMemory(MB)</th>
                                <th data-options="field:'memoryFreePt'" width="80">FreeMemoryPt(%)</th>
                                <th data-options="field:'memoryUsed'" width="80">UsedMemory(MB)</th>
                                <th data-options="field:'memoryUsedPt'" width="80">UsedMemoryPt(%)</th>

                            </tr>
                            </thead>
                        </table>
                    </div>
                    <div region="center" data-options="collapsible:false,border:false" title="Disk">
                        <table class="easyui-datagrid" fit="true" fitColumns="true"
                               data-options="border:false,
                               singleSelect:true,
                               url: '${ctx}/system/resourceMonitor/getFileSystemInfo.action',
                               striped : true,onLoadSuccess:datagridLoadSuccess">
                            <thead>
                            <tr>
                                <th data-options="field:'devName'" width="80">DevName</th>
                                <th data-options="field:'sysTypeName'" width="80">TypeName</th>
                                <th data-options="field:'sysTotalSize'" width="80">TotalSize(GB)</th>
                                <th data-options="field:'sysFreeSize'" width="80">FreeSize(GB)</th>
                                <th data-options="field:'sysUsedSize'" width="80">UsedSize(GB)</th>
                                <th data-options="field:'sysUsedPercent'" width="80">UsagePercent</th>
                            </tr>
                            </thead>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>