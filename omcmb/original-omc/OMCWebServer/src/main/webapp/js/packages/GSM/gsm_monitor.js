Vue.component('gsm-monitor', {
    template: `
    <div id="gsmMonitorPage" class="monitor-ctner" style="overflow: auto;height: 100%;">
        <el-ctable ref="list" height="100%"
            id="gsmTableHomeCellList"
            row-key="serial_number"
            :url="tbURL"
            :time="6"
            :query-params="queryParams"
            :limit="limitBatch" 
            @load-success="loadSuccess"
            @selection-change="selectionChange">
            <template slot="toolbar">
                <div class="toolbarHeadBtnBoxCls">
                    <div id="addGsmDeviceOrImport" @click="showAddOrImport" style="position: absolute;right: 0px;top: 0px;">
                        <!-- <el-popover trigger="click" placement="bottom-end" popper-class="monitorBtnPopperCls">
                            <div class="el-card__header">
                                Add
                                <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                            </div>
                            <div class="addGsmOrImport-content" style="width: 700px; max-height: 500px;overflow: auto;"></div>
                            <div slot="reference" class="newIconBoxCls-bt CODE_ENB_DEVICE_REGISTER hidden" style="right:56px;top:5px;" tip="Add">
                                <span class="el-icon-plus el-icon"></span>
                            </div>	
                        </el-popover> -->
                        <div  v-if="operatorShow" @click="showGsmExport">
                            <el-popover trigger="click" placement="bottom-end" popper-class="monitorBtnPopperCls">
                                <div class="el-card__header">
                                    {{local.export}}
                                    <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                                </div>
                                <div style="width: 920px; max-height: 500px;overflow: auto;">
                                    <el-ctable ref="operatorCtable" :url="operatorCtableUrl" row-key="operator_code" style="margin: 0px 20px 20px 20px;border: 1px solid #f3f3f3;" height="200" :default-checked="operatorCtableDefaultCheckList" @selection-change="operatorCtableHandleChange">
                                        <el-table-column prop="operator_code" type="selection" :reserve-selection="true"></el-table-column>
                                        <el-table-column prop="operator_name" :label="local.operator"></el-table-column>
                                    </el-ctable>
                                    <div style="margin-left: 20px;">
                                        <div style="font-size: 14px;color:#606266;font-weight:bold;">{{local.licenseInfo}}</div>
                                        <el-checkbox v-model="isExportDeviceLicenseInfo" style="margin: 10px 20px;">{{local.exportLicense}}</el-checkbox> 
                                    </div>
                                    <div style="padding: 20px;">
                                        <el-button type="primary" @click="comfirmExport">{{local.ok}}</el-button>
                                        <el-button @click="close">{{local.cancel}}</el-button>
                                        <span style="color: 999;font-weight: normal;margin-left: 20px;"><i class="el-icon el-icon-circle-info gray-color"></i> Waiting</span>
                                    </div>
                                </div>
                                <div slot="reference" class="newIconBoxCls-bt" style="right:20px;top:5px;" :tip="local.export">
                                    <span class="el-icon-operation-export el-icon"></span>
                                </div>	
                            </el-popover>
                        </div>
                        <el-dropdown v-if="!operatorShow" @command="exportCSV" trigger="click" class="exportDrop">
                            <div class="newIconBoxCls-bt" style="right:0px;top:-5px;" :tip="local.export">
                                <span class="el-icon-operation-export el-icon"></span>
                            </div>	
                            <el-dropdown-menu slot="dropdown">
                                <el-dropdown-item command="monitor">{{local.exportMonitor}}</el-dropdown-item>
                                <el-dropdown-item command="license">{{local.exportLicense}}</el-dropdown-item>
                            </el-dropdown-menu>
                        </el-dropdown>
                       
                    </div>
                    <div class="selectBlukBoxCls">
                        <div class="selectMain">
                            <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                <span class="el-icon-selected el-icon"></span>
                                <span class="bulkSelectNumBoxCls">( {{selectedRows.length}} )</span>
                            </div>
                            <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                                <div class="selectBoxTitle">
                                    <span>{{local.selected}}</span>
                                    <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                </div>
                                <div class="selectBoxMain">
                                    <div class="tableInfoCls">
                                        <div class="tableInfoHeader">
                                            <div>{{local.deviceCode}}</div>
                                            <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
                                        </div>
                                        <el-ctable 
                                            id="gsmBulkSelectTable" 
                                            ref="bulkSelectTable" 
                                            :data="selectedRows" 
                                            :showHeader="false"
                                            :rownumber="false"
                                            :front-pagination="true"
                                            height="300px" pagination="true" >
                                            <el-table-column prop="id" v-if="false"></el-table-column>
                                            <el-table-column width="588">
                                                <template slot-scope="scope" >
                                                    <div class="tableItemCls">
                                                        <span>{{scope.row.serial_number}}</span>
                                                        <span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
                                                    </div>
                                                </template>
                                            </el-table-column>
                                        </el-ctable>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="movecells">
                        <span class="el-icon el-icon-moveGroup"></span>
                        <span>{{local.moveGroup}}</span>
                    </div>
                    <div :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="syncList">
                        <span class="el-icon el-icon-operation-synchronize"></span>
                        <span>{{local.sync}}</span>
                    </div>
                    <div :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="rebootList">
                        <span class="el-icon el-icon-operation-reboot"></span>
                        <span>{{local.reboot}}</span>
                    </div>
                </div>
                <gsm-query :local="gsmQueryLocal" :ctx="ctx"></gsm-query>
            </template>

            <el-table-column v-if="enableCheckbox" key="selection" type="selection" :selectable="selectable" :reserve-selection="true" fixed></el-table-column>
            <el-table-column prop="enb_operation" width="70" fixed>
                <template slot-scope="scope">
                    <div class="el-icon el-icon-circle-setting1" @click="openSettingPage(scope.row)" style="font-size:19px;"></div>
                    <div class="el-icon el-icon-operation-more-circle" @click="optClick(scope.row, event)" v-clickoutside="handerClose"></div>
                </template>
            </el-table-column>
            <el-table-column sortable prop="connection_status" key="connection_status" width="40" fixed>
                <template slot-scope="scope">
                    <div v-html="connStatusFmt(scope.row, scope.row['connection_status'], scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column prop="alarm_count" key="alarm_count" width="110" :label="local.alarm" sortable fixed>
                <template slot-scope="scope">
                    <div style="text-align: center;">
                        <span v-if="scope.row.alarm_serverity == '31001' "  class="alarmCritical alarmListSty" @click="alarmToInfo(scope.row, event)">{{scope.row.alarm_count}}</span>
                        <span v-else-if="scope.row.alarm_serverity == '31002' "  class="alarmMajor alarmListSty" @click="alarmToInfo(scope.row, event)">{{scope.row.alarm_count}}</span>
                        <span v-else-if="scope.row.alarm_serverity == '31003' "  class="alarmMinor alarmListSty" @click="alarmToInfo(scope.row, event)">{{scope.row.alarm_count}}</span>
                        <span v-else-if="scope.row.alarm_serverity == '31004' "  class="alarmWarning alarmListSty" @click="alarmToInfo(scope.row, event)">{{scope.row.alarm_count}}</span>
                        <span v-else >0</span>
                    </div>
                </template>
            </el-table-column>
            <el-table-column key="serial_number" sortable :label="local.bscCode" prop="serial_number" width="180" fixed></el-table-column>
            <el-table-column key="host_name" sortable :label="local.bscName" prop="host_name" width="150" fixed>
                <template slot-scope="scope">
                    <div v-if="scope.row.device_name_tip === '0'">{{scope.row.host_name}}</div>
                    <div v-if="scope.row.device_name_tip !== '0'" class="cellNameClass">
                        <el-popover trigger="click">
                            <div slot="reference">
                                <span class="el-icon el-icon-circle-warning"></span> 
                                {{scope.row.host_name}}
                            </div>
                            <div style='padding: 30px 20px 20px; position:relative;'>
                                <div> <span class="el-icon el-icon-close" @click="closeSyncName" style="top: 10px; position:absolute;"></span>
                                <div style='display: flex;'><span style='color: #333; font-size: 14px;'>{{local.enbName}} : </span> <span style='font-size: 14px;margin-left: 6px;color: #333;'>{{scope.row.report_host_name}}</span></div>
                                <div style='font-size: 12px; color: #333; margin-top: 6px;'>{{local.sysncToOMC}}<div>
                                <div style="margin-top: 10px; text-align: right;">
                                    <span class="el-button--primary el-button" @click="syncName(scope.row.small_cell_code, scope.row.report_host_name)">
                                        {{local.ok}}
                                    </span>
                                    <span class="white el-button" @click="closeSyncName">{{local.cancel}}</span>
                                </div>
                            </div>
                        </el-popover>
                    </div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('rf_status')" key="rf_status" sortable :label="local.rfStatus" prop="rf_status" width="150" :show-overflow-tooltip="false">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'" style="display: flex;align-items: center;">
                        <div v-if="['','NULL','null',null,undefined].includes(scope.row.rf_status)"></div>
                        <div v-if="scope.row.rf_status == '--'">--</div>
                        <div v-if="!['','NULL','null',null,undefined].includes(scope.row.rf_status) && scope.row.rf_status != '--'">
                            <div v-if="scope.row.rf_status == 'on' && parseCellAndRfStatus(scope.row.rf_status).length == 1" class='iconFlexCls'>
                                <span style='margin-right: 5px;'>{{local.on}}</span>
                            </div>
                            <div v-if="scope.row.rf_status == 'off' && parseCellAndRfStatus(scope.row.rf_status).length == 1" class='iconFlexCls'>
                                <span class='offStatusCls' style='margin-right: 5px;'>{{local.off}}</span>
                            </div>
                            <div v-if="parseCellAndRfStatus(scope.row.rf_status).length > 1 "  style="display: flex;">
                                <span v-if="judgeActiveStatusFat(scope.row.rf_status) == '1'" style='margin-right: 5px;'>{{local.on}}</span>
                                <span v-if="judgeActiveStatusFat(scope.row.rf_status) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'>{{local.on}}</span>
                                <span v-if="judgeActiveStatusFat(scope.row.rf_status) == '3'" class='offStatusCls' style='margin-right: 5px;'>{{local.off}}</span>
                                <el-popover :title="local.multCellStatus" popper-class="cellActivePopoverClass" trigger="click" width='300'>
                                    <span style="color:#4d84ff;" slot="reference" v-if="parseCellAndRfStatus(scope.row.rf_status).length>1">
                                        [ {{judgeActiveNumOrAllNumFat(scope.row.rf_status,'active')}}/{{judgeActiveNumOrAllNumFat(scope.row.rf_status,'all')}} ]
                                    </span>
                                    <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
                                        <div v-for="(item,index) in parseCellAndRfStatus(scope.row.rf_status)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
                                            Cell {{index+1}}:
                                            <span v-if=" item == 'on'" class="onStatusBoxCls" style='margin-left: 5px;'>{{local.on}}</span>
                                            <span v-if=" item == 'off'" class="offStatusBoxCls" style='margin-left: 5px;'>{{local.off}}</span>
                                        </div>
                                    </div>
                                </el-popover>
                            </div>
                        </div>
                    </div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('op_state')" key="op_state" sortable :label="local.activeStatus" prop="op_state" width="120">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'" style="display: flex;align-items: center;">
                        <div v-if="scope.row.product != 'PM-B4860'">
                            <div v-if="scope.row.op_state == '1' && !['Intel_CR_CA','Intel_CR_TC'].includes(scope.row.platformType)" class='iconFlexCls'>
                                <span style='margin-right: 5px;'>{{local.active}}</span>
                            </div>
                            <div v-if="scope.row.op_state == '0' && !['Intel_CR_CA','Intel_CR_TC'].includes(scope.row.platformType)" class='iconFlexCls'>
                                <span class='offStatusCls' style='margin-right: 5px;'>{{local.inactive}}</span>
                            </div>
                            <div v-if="['Intel_CR_CA','Intel_CR_TC'].includes(scope.row.platformType)"  style="display: flex;">
                                <span v-if="judgeActiveStatusFat(scope.row.op_state) == '1'" style='margin-right: 5px;'>{{local.active}}</span>
                                <span v-if="judgeActiveStatusFat(scope.row.op_state) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'>{{local.active}}</span>
                                <span v-if="judgeActiveStatusFat(scope.row.op_state) == '3'" class='offStatusCls' style='margin-right: 5px;'>{{local.inactive}}</span>
                                <el-popover :title="local.multCellStatus" popper-class="cellActivePopoverClass" trigger="click" width='300'>
                                    <span style="color:#4d84ff;" slot="reference">
                                        [ {{judgeActiveNumOrAllNumFat(scope.row.op_state,'active')}}/{{judgeActiveNumOrAllNumFat(scope.row.op_state,'all')}} ]
                                    </span>
                                    <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
                                        <div v-for="(item,index) in parseCellAndRfStatus(scope.row.op_state)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
                                            Cell {{index+1}}:
                                            <span v-if=" item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'>{{local.active}}</span>
                                            <span v-if=" item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'>{{local.inactive}}</span>
                                        </div>
                                    </div>
                                </el-popover>
                            </div>
                            <div v-if="['QA_436Q_CA'].includes(scope.row.platformType)"  style="display: flex;">
                                <span v-if="judgeActiveStatusFat(scope.row.op_state) == '1'" style='margin-right: 5px;'>{{local.active}}</span>
                                <span v-if="judgeActiveStatusFat(scope.row.op_state) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'>{{local.active}}</span>
                                <span v-if="judgeActiveStatusFat(scope.row.op_state) == '3'" class='offStatusCls' style='margin-right: 5px;'>{{local.inactive}}</span>
                                <el-popover :title="local.multCellStatus" popper-class="cellActivePopoverClass" trigger="click" width='300'>
                                    <span style="color:#4d84ff;" slot="reference">
                                        [ {{judgeActiveNumOrAllNumFat(scope.row.op_state,'active')}}/{{judgeActiveNumOrAllNumFat(scope.row.op_state,'all')}} ]
                                    </span>
                                    <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
                                        <div v-for="(item,index) in parseCellAndRfStatus(scope.row.op_state)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
                                            Cell {{index+1}}:
                                            <span v-if=" item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'>{{local.active}}</span>
                                            <span v-if=" item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'>{{local.inactive}}</span>
                                        </div>
                                    </div>
                                </el-popover>
                            </div>
                        </div>
                        <div v-if="scope.row.product == 'PM-B4860'">
                            <div class='iconFlexCls'>
                                <span v-if="scope.row.op_state == '0'" class='offStatusCls' style='margin-right: 5px;'>{{local.inactive}}</span>
                                <span v-if="scope.row.op_state != '0'" class='offOrOnStatusCls' style='margin-right: 5px;'>{{local.active}}</span>
                                <el-popover :title="local.multCellStatus" popper-class="cellActivePopoverClass" trigger="click" width='640' @show="cellActivePopoverShow(scope.row)" @hide="cellActivePopoverHide">
                                    <span style="color:#4d84ff;" slot="reference">
                                        [ {{scope.row.op_state}}/{{scope.row.op_state_cell2}} ]
                                    </span>
                                    <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:120px;" v-loading="cellActivePopoverLoading" >
                                        <div v-for="item in activeCellsDataList" v-show='item.cellItem.length>0' style="height:40px;display:flex;align-items: center;margin-bottom:10px;">
                                            <div style="width:110px;">{{item.label}}（{{local.bandId}}）</div>
                                            <div v-for="(items,index) in item.cellItem" style="padding-left:10px;height:40px;width:160px;display:flex;align-items: center;justify-content: center;border:1px solid #E9E9E9;">
                                                <div v-if="items.activeStatus == '1'">
                                                    <span style="margin-right: 5px;">{{items.cellIndex}}</span>
                                                    <span class="onStatusBoxCls">{{local.active}}</span>
                                                    
                                                </div>
                                                <div v-if="items.activeStatus == '0'">
                                                    <span style="margin-right: 5px;">{{items.cellIndex}}</span>
                                                    <span class="offStatusBoxCls">{{local.inactive}}</span>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </el-popover>
                            </div>
                        </div>
                        <el-tooltip v-if="scope.row.delay_avaliable == '0' || scope.row.delay_avaliable == '1'" :content="local.licenseExpired">
                            <span class="warnTips el-icon el-icon-sas-warning" style='margin-left: 5px;'></span>
                        </el-tooltip>
                    </div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('product')" key="product" sortable :label="local.product" prop="product" width="110">
                <template slot-scope="scope">
                    <div v-html="productFmt(scope.row, scope.row.product, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('module_type')" key="module_type" sortable :label="local.module" prop="module_type" width="120">
                <template slot-scope="scope">
                    <div v-html="capablityFmt(scope.row, scope.row.module_type, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('product_name')" key="product_name" sortable :label="local.productName" prop="product_name" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('cell_ip')" key="cell_ip" sortable :label="local.cellIp" prop="cell_ip" width="120">
                <template slot-scope="scope">
                    <div v-html="ipAddrFmt(scope.row, scope.row.cell_ip, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('sub_station_name')" key="sub_station_name"  :label='local.siteName' prop="sub_station_name" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('mac_address')" key="mac_address" sortable label="MAC" prop="mac_address" width="130"></el-table-column>
            <el-table-column v-if="showColums.includes('ue_count')" key="ue_count" sortable :label="local.ueCount" prop="ue_count" width="80">
                <template slot-scope="scope">
                    <div v-html="ueCountsFormatter(scope.row, scope.row.ue_count, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('IpaUnitId')" key="IpaUnitId" label="Ipa Unit Id" prop="IpaUnitId" width="120">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.IpaUnitId}}</div>
                    <div v-if="scope.row.product == 'BSC'">--</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('OmlRemoteIp')" key="OmlRemoteIp" label="Oml Remote Ip" prop="OmlRemoteIp" width="120">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.OmlRemoteIp}}</div>
                    <div v-if="scope.row.product == 'BSC'">--</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('OmlRemoteIpBak')" key="OmlRemoteIpBak" label="Oml Remote Ip Bak" prop="OmlRemoteIpBak" width="150">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.OmlRemoteIpBak}}</div>
                    <div v-if="scope.row.product == 'BSC'">--</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('BscSelect')" key="BscSelect" label="BSC Select" prop="BscSelect" width="100">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BTS'">
                        <span v-if="scope.row.BscSelect == '0'">{{local.main}}</span>
                        <span v-if="scope.row.BscSelect == '1'">{{local.sub}}</span>
                    </div>
                    <div v-if="scope.row.product == 'BSC'">--</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('BscLinkStatus')" key="BscLinkStatus" :label="local.bscLinkStatus" prop="BscLinkStatus" width="120">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BTS'">
                        <span v-if="scope.row.BscLinkStatus == '0'">{{local.conn}}</span>
                        <span v-if="scope.row.BscLinkStatus == '1'">{{local.disconn}}</span>
                    </div>
                    <div v-if="scope.row.product == 'BSC'">--</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('BSCSerialNumber')" key="BSCSerialNumber" :label="local.bscCodeRef" prop="BSCSerialNumber" width="160">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.BSCSerialNumber}}</div>
                    <div v-if="scope.row.product == 'BSC'">--</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('BtsNum')" key="BtsNum" :label="local.btsNum" prop="BtsNum" width="90">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BTS'">--</div>
                    <div v-if="scope.row.product == 'BSC'">{{scope.row.BtsNum}}</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('synStatus')" key="synStatus" sortable :label="local.syncStatus" prop="synStatus" width="160">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'" v-html="syncStatusFmt(scope.row, scope.row.synStatus, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('gps_satellite_count')" key="gps_satellite_count" sortable :label="local.satellite" prop="gps_satellite_count" width="85">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'">
                        <div v-if='scope.row.hasSatelliteDetail == "true"'>
                            <a style='color:#1DA3FC;text-decoration:underline' href='#' @click='getGsmGPSSignalData(scope.row)'>{{scope.row.gps_satellite_count}}</a>
                        </div>
                        <div v-if='scope.row.hasSatelliteDetail != "true"'> {{scope.row.gps_satellite_count}} </div>
                    </div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('gps_longitude')" key="gps_longitude" :label="local.longitude" prop="gps_longitude" width="100">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.gps_longitude}}</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('gps_latitude')" key="gps_latitude" :label="local.latitude" prop="gps_latitude" width="90">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.gps_latitude}}</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('gps_height')" key="gps_height" :label="local.height" prop="gps_height" width="70">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.gps_height}}</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('currentLac')" key="currentLac" label="LAC" prop="currentLac" width="70">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.currentLac}}</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('currentArfcn')" key="currentArfcn" :label="local.arfcn" prop="currentArfcn" width="70">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'">{{scope.row.currentArfcn}}</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('uplinkFrequency')" key="uplinkFrequency" :label="local.upfreq" prop="uplinkFrequency" width="110">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-else>
                        {{scope.row.uplinkFrequency}}
                        <span v-if="scope.row.uplinkFrequency">MHz</span>
                    </div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('downlinkFrequency')" key="downlinkFrequency" :label="local.downfreq" prop="downlinkFrequency" width="110">
                <template slot-scope="scope">
                    <div v-if="scope.row.product == 'BSC'">--</div>
                    <div v-if="scope.row.product == 'BTS'">
                        {{scope.row.downlinkFrequency}}
                        <span v-if="scope.row.downlinkFrequency">MHz</span>
                    </div>
                </template>
            </el-table-column>
            
            <el-table-column v-if="showColums.includes('up_time')" key="up_time" sortable :label="local.uptime" prop="up_time" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('online_duration')" key="online_duration" sortable :label="local.duration" prop="online_duration" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('first_online_time')" key="first_online_time" sortable :label="local.firstTime" prop="first_online_time" width="140"></el-table-column>
            <el-table-column v-if="showColums.includes('LASTINFORMTIME')" key="LASTINFORMTIME" sortable :label="local.lastTime" prop="LASTINFORMTIME" width="220"></el-table-column>
            <el-table-column v-if="showColums.includes('software_version')" key="software_version" sortable :label="local.software" prop="software_version" width="140"></el-table-column>
            <el-table-column v-if="showColums.includes('firmware_version')" key="firmware_version" sortable :label="local.firmware" prop="firmware_version" width="135"></el-table-column>
            <el-table-column v-if="showColums.includes('group_name')" key="group_name" sortable :label="local.group" prop="group_name" width="130"></el-table-column>
        </el-ctable>
        <el-cmenu ref="menu" :data="menus" @click="menuClick"></el-cmenu>
        <div style="position: absolute;bottom: 15px;z-index: 100;display: flex;right: 100px;">
            <div class="status-statistic active" style="border-right: none;">
                <span class="el-icon el-icon-status-conn-on"></span>
                <span style="margin: 0 10px;">{{local.isOnline}}</span> 
                <span id="gsm_online_count_rate" style="margin-left: 15px;"></span>
            </div>
            <!-- <div class="status-statistic active" style="border-right: none;">
                <span class="el-icon el-icon-status-active"></span>
                <span style="margin: 0 10px;">{{local.activeStatus}} </span> 
                <span id="gsm_active_count_rate" style="margin-left: 15px;"></span>
            </div> -->
        </div>

		<el-slide ref="settingPage" class="settingSlide" 
           	:url="settingUrl" width="80%"
            :footer="false" 
            :header="false">
        </el-slide>
		<el-slide ref="satelite" class="no-padding"
            :title='local.satellite'
            @cancel="closeSatellite"
            :footer="false" >
            <el-ctable :url="satelliteURL" :class="{loading: sateloading}">
                <el-table-column :label='local.gnssId' prop='gnssSvId'></el-table-column>
                <el-table-column :label='local.snr + "(dB-Hz)"' prop='snr'></el-table-column>
            </el-ctable>
        </el-slide>
        <!-- sliders -->
        <el-slide ref="ueCount" :title="ueslide.title" :height='ueslide.height' :footer="false" @cancel="closeUeSlide" class='commonWarp'>
            <el-ctable :data="ueslide.data" :pagination="false">
                <el-table-column label="UEID" prop="ue_id" width="100"></el-table-column>
                <el-table-column label="IMSI" prop="imsi" width="150"></el-table-column>
                <el-table-column label="VMAC" prop="vmac" width="130"></el-table-column>
                <el-table-column :label="local.cpeName" prop="cpe_name" width="150"></el-table-column>
                <el-table-column :label="local.downRate" width="140" prop="downlink_rate"></el-table-column>
                <el-table-column :label="local.upRate" width="135" prop="uplink_rate"></el-table-column>
                <el-table-column :label="local.ip" prop="ip" width="140"></el-table-column>
                <el-table-column :label="local.port" prop="port" width="80"></el-table-column>
                <el-table-column :label="local.ulsinr" prop="ulsinr" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" :label="local.dlcqi" prop="dlcqi" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_Dlcqi" prop="p_dlcqi" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_Dlcqi" prop="s_dlcqi" width="100"></el-table-column>

                <el-table-column :label="local.ulmcs" prop="ulmcs" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" :label="local.dlmcs" prop="dlmcs" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_Dlmcs" prop="p_dlmcs" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_Dlmcs" prop="s_dlmcs" width="100"></el-table-column>

                <el-table-column :label="local.txpower + "(dBm)"' prop="txpower" width="100"></el-table-column>
                <el-table-column :label="local.upbler + "(%)"' prop="uplink_bler" width="120"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_TB1_Downlink_BLER(%)" prop="p1_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="P_TB2_Downlink_BLER(%)" prop="p2_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_TB1_Downlink_BLER(%)" prop="s1_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_TB2_Downlink_BLER(%)" prop="s2_downlink_bler" width="165"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" :label="local.downbler + "(%)"' prop="downlink_bler" width="165"></el-table-column>

                <el-table-column :label="local.path + "(dBm)"' prop="pathloss" width="120"></el-table-column>
                <el-table-column v-if='ueS1apId == true' label="UE_S1AP_ID" prop="ue_s1ap_id" width="120"></el-table-column>
				<el-table-column v-if='mmeS1apId == true' label="MME_S1AP_ID" prop="mme_s1ap_id" width="120"></el-table-column>
            </el-ctable>
        </el-slide>

        <!--激活/取消激活 弹窗-->
		<el-dialog :title="local.confirm" id="gsmActiveCellDialog" :visible.sync="showActiveCellDialog" top="30vh" ref="activeCellDialog" 
			width="660" :close-on-click-modal="false"  @close='closeActiveCellDialog' append-to-body>
				<div style="margin-bottom:20px;">确认激活/取消激活？</div>
				<div style="border:1px solid #e9e9e9;padding:10px;min-height:100px;" v-loading='activeCellLialogLoading'>
						<div v-for="item in cellInfosList" class="borderCardItemCls" v-show='item.cellItem.length>0'>
							<span class="borderCardItemLabelCls">{{item.label}}（{{local.bandId}）</span>
							<div v-for="(items,index) in item.cellItem"  style="display:inline-block;margin-right:10px;width:150px;font-size:12px;">
								<el-checkbox v-model="items.activeStatus" :true-label="1" :false-label="0">
									{{index+1}}
								</el-checkbox>
								<div v-if="items.oldStatus == '0'" style="display:inline-block;">
									(<span class="el-icon el-icon-status-active redIcon"></span>{{local.inactive}})
								</div>
								<div v-if="items.oldStatus == '1'" style="display:inline-block;">
									(<span class="el-icon el-icon-status-active greenIcon"></span>{{local.active}})
								</div>
							</div>
						</div>
				</div>
				<span slot="footer">
					<div>
						<el-button type="primary" @click="activeCellSubmit">{{local.ok}}</el-button>
						<el-button @click="closeActiveCellDialog">{{local.cancel}}</el-button>
					</div>
				</span>
		</el-dialog>

        <el-dialog :title="local.confirm" top="30vh" width="550"
            :visible.sync="collectMessageShow" 
            :modal="false"
            :close-on-click-modal="false">
            <el-form :model="collectForm">
                <div>{{confirmTips}}</div>
                <el-form-item :label="local.interval" style="display: flex;align-items: center;margin: 5px 0px;">
                    <el-select v-model="collectForm.collectInterval" placeholder="Select time" class="collect-select">
                        <el-option label="05" value="05"></el-option>
                        <el-option label="10" value="10"></el-option>
                    </el-select>
                    <div style="display: inline;padding: 5px;margin-left: -4px;border: 1px solid #e9e9e9;background: #F5F7FA;">{{local.anrTime}}</div>
                </el-form-item>
                <span v-if="cllectExisted">
                    <span style="color: #B3B3B3;">{{local.collectTip}} SN={{existedMsgSN}}. </span>
                </span>
            </el-form>

            <div slot="footer" style="text-align: right;">
                <el-button type="primary" @click="sendCollect">{{local.ok}}</el-button>
                <el-button @click="collectMessageShow = false">{{local.cancel}}</el-button>
            </div>
        </el-dialog>
        
        <el-dialog :title="local.moveGroup" :visible.sync="deviceGroupMoveShow" width="620"
			:close-on-click-modal="false" top="30vh" :append-to-body="true">
            <el-ctable ref="ctableGroup" :url="deviceGroupUrl" id='gsmCtableGroup' @row-click="groupIdChange" :height="groupHeight" :row-key="'id'" pagination="true" :rownumber=true style='border: 1px solid #E9E9E9;'>
				<el-table-column label='' width="40">
					<template slot-scope="scope">
		           		<el-radio v-model="groupId" :label="scope.row.id"><span></span></el-radio>
		         	</template>
				</el-table-column>
				<el-table-column :label="local.groupName" min-width="150" prop="group_name"></el-table-column>
			</el-ctable>
			<div v-show="deviceGroupTip"><div slot="tip" class="el-upload__tip" >{{local.selectGroup}} </div></div>
            <div slot="footer" style='text-align: right;'>
                <el-button type="primary" @click="deviceGroupMoveSubmit">{{local.ok}}</el-button>
                <el-button @click="deviceGroupMoveShow = false">{{local.cancel}}</el-button>
            </div>
        </el-dialog>

        <!-- 参数同步弹窗 -->
        <el-dialog :title="local.sync" :visible.sync="openSyncDialogShow" :close-on-click-modal="false" @close="closeSyncDialog">
            <div id="gsm_sync-content" :class="{'loading': isSyncloading}" style="min-height: 200px;">
            </div>
        </el-dialog>
    </div>
    `,
    el: '#gsmMonitorPage',
    props: {
        ctx: {
            type: String,
            default: ''
        },
        local: {
            type: Object,
            default() {
                return {
                    info: 'Information',
                    export: 'Export',
                    operator: 'Operator Name',
                    licenseInfo: 'License',
                    exportLicense: 'Export License',
                    exportMonitor: 'Export eNB',
                    ok: 'OK',
                    cancel: 'Cancel',
                    selected: 'Selected',
                    deviceCode: 'Serial Number',
                    moveGroup: 'Move to group',
                    sync: 'Sync',
                    reboot: 'Reboot',
                    alarm: 'Alarm',
                    bscCode: 'BSC Code',
                    bscName: 'BSC Name',
                    enbName: 'Site Name',
                    sysncToOMC: 'Sync to OMC',
                    rfStatus: 'RF Status',
                    on: 'On',
                    off: 'Off',
                    multCellStatus: 'Multi Cell Status',
                    activeStatus: 'Active Status',
                    active: 'Active',
                    inactive: 'Inactive',
                    bandId: 'Band ID',
                    licenseExpired: 'License Expired',
                    product: 'Product',
                    productName: 'Product Name',
                    cellIp: 'IP',
                    module: 'Module',
                    siteName: 'Site Name',
                    ueCount: 'UE Count',
                    main: 'Main',
                    sub: 'Sub',
                    bscLinkStatus: 'Connection',
                    conn: 'Connected',
                    disconn: 'Disconnected',
                    bscCodeRef: 'BSC SerialNumber',
                    btsNum: 'BTS Num',
                    syncStatus: 'Sync Status',
                    satellite: 'Satellites',
                    longitude: 'Longitude',
                    latitude: 'Latitude',
                    height: 'Height',
                    arfcn: 'ARFCN',
                    upfreq: 'UP Freq',
                    downfreq: 'Down Freq',
                    uptime: 'Run Time',
                    duration: 'Duration',
                    firstTime: 'First Online Time',
                    lastTime: 'LASTINFORMTIME',
                    software: 'Software Version',
                    firmware: 'Hardware Version',
                    group: 'Group',
                    gnssId: 'ID',
                    snr: 'SNR',
                    cpeName: 'CPE Name',
                    downRate: 'Downlink Rate',
                    upRate: 'Uplink Rate',
                    ip: 'IP',
                    port: 'Port',
                    ulsinr: 'Ulsinr',
                    dlcqi: 'Dlcqi',
                    ulmcs: 'Ulmcs',
                    dlmcs: 'Dlmcs',
                    txpower: 'Txpower',
                    upbler: 'Uplink Bler',
                    downbler: 'Downlink Bler',
                    path: 'Pathloss',
                    confirm: 'Confirm',
                    interval: 'Interval',
                    anrTime: '',
                    collectTip: '',
                    groupName: 'Group Name',
                    selectGroup: '',
                    detail: 'Detail',
                    collectPre: '',
                    collecting: '',
                    success: 'Success',
                    rebootTip: '',
                    sendTip: '',
                    setting: 'Setting',
                    notselectall: '',
                    collectLog: 'Logs',
                    modifyPWD: 'Reset Password',
                    license: 'License',
                    operation: 'Operation',
                    registerSAS: 'Register SAS',
                    deregisterSAS: 'Deregister SAS',
                    moreOpt: 'More',
                    maintaince: 'Maintainse',
                    restore: 'Restore',
                    rfName: 'RF Name',
                    validDate: '',
                    limitation: 'Limitation',
                    dispatch: 'Dispatch',
                    collectReport: '',
                    logCollecting: '',
                    inSync: '',
                    syncSuccess: '',
                    notSync: '',
                }
            }
        },
        carrierType: {
            type: String,
            default: ''
        }
    },
    data() {
        var vm = this;

        return {
            isSyncloading:false,
            isScanloading: false,
            scanChart: null,
            scanEnable: '0',
            scanForm: {
                mode: '0'
            },

            cllectExisted: false,
            existedMsgSN: '',
            collectMessageShow: false,
            collectForm: {
                collectInterval: ''
            },

            sortColumns: [],
            columns: [
                {field: 'serial_number', label: this.local.bscCode,sortable: true, width: 180},
                {field: 'host_name', label: this.local.bscName,sortable: true, width: 150},
                {field: 'op_state', label: this.local.activeStatus,sortable: true, width: 120},
                {field: 'product', label: this.local.product,sortable: true, width: 110},
                {field: 'module_type', label: this.local.module,sortable: true, width: 120},
                {field: 'product_name', label: this.local.productName,sortable: true, width: 120},
                {field: 'cell_ip', label: this.local.cellIp,sortable: true, width: 120},
                {field: 'sub_station_name', label: this.local.siteName,width: 120},
                {field: 'mac_address', label: 'MAC',sortable: true, width: 130},
                {field: 'ue_count', label: this.local.ueCount,sortable: true, width: 80},
                {field: 'up_time', label: this.local.uptime,sortable: true, width: 120},
                {field: 'online_duration', label: this.local.duration,sortable: true, width: 120},
                {field: 'first_online_time', label: this.local.firstTime,sortable: true, width: 140},
                {field: 'LASTINFORMTIME', label: this.local.lastTime,sortable: true, width: 220},
                {field: 'software_version', label: this.local.software,sortable: true, width: 140},
                {field: 'firmware_version', label: this.local.firmware,sortable: true, width: 135},
                {field: 'group_name', label: this.local.groupName,sortable: true, width: 130},

            ],
            selectedRow: '',
            menus: [],
            settingUrl:'',
            tbURL: this.ctx + '/cell/cpeinfos/queryCpeInfosList.action?monitor=1',
            queryParams: {
                TimeZone : timeZone,
                isDual: false,
                isMonitor: true,
                search_text: '',
                like_fields: 'serial_number,host_name,cell_ip',
                connection_status: [],
                op_state: '',
                product_model: 'BSC,BTS',
                model_name: [],
                software_version: [],
                firmware_version: [],
                halob_flag: '',
                group_id: [],
                halodSerialNumbers:'',
                bscSerialnumber:''
            },

            showProps: enbShowCols.split(','),
            enableCheckbox: (writableMap['CODE_ENB_REBOOT'] == true || writableMap['CODE_ENB_SYNCHRONIZE'] == true),
            selectedRows: [],

            infoslide: {
                title: this.local.info,
                url: ''
            },
            ueslide: {
                title: '',
                height: '',
                data: '',
                is436q: false
            },
            cpeslide: {
                title: '',
                url: '',
            },
            activeslide: {
                title: this.local.detail,
                data: ''
            },
            periodslide: {
                title: '',
                url: ''
            },
            distributeslide: {
                title: '',
                url: ''
            },
            enbSettingSlide: {
                title: '',
                url: ''
            },
            satelliteURL: '',
            sateloading: true,
            activeCells:[],
            cellInfosList:[],
            showActiveCellDialog:false,
            activeCellLialogLoading:false,
            activeCellsDataList:[],
            cellActivePopoverLoading:false,
            deviceGroupMoveShow: false,
            groupId: '',
            deviceGroupUrl: this.ctx + '/pm/template/getTempDeviceGroupList.action',
            groupHeight:'320px',
            deviceGroupTip:false,
            halodCellPopoverLoading:false,
            halodCellDataList:[],
            lockForm: {
                lockStatus: '0',
            },
            lockReason: '',
            lockMac: '',
            curRowSn: '',
            addLoading: false,
            lockReasonShow: false,
            bulkSelectShow:false,
            
            ueS1apId: false,
            mmeS1apId: false,

            openSyncDialogShow:false,
            batchSync:false,
            batchCode:'',

            operatorCtableDefaultCheckList: [],
            operatorCtableCheckList:[],
            operatorCtableUrl: this.ctx + '/system/operator/getOperatorListByPage.action',
            isExportDeviceLicenseInfo: false,
            
        };
    },
    computed: {
        limitBatch(){
            return batchOperation ? '' : 1;
        },
        confirmTips() {
            var vm = this,
                sn = vm.selectedRow.serial_number,
                msg = this.local.collectPre;

            return msg.replace('placeholder', sn);
        },
        showColums() {
            var vm = this,
                columns = vm.getAllDefaultCols(),
                props = columns.map(function(col){
                    return col.prop;
                });

            //props = props.filter(function(code){
                // return vm.showProps.includes(code);
            //});

            return props;
        },
        operatorShow() {
            return isCloud == 'true' && is_super_user == 'true';
        },
        gsmQueryLocal() {

            return gsmQueryLocal;
        },
        context() {
            
            return this.ctx || '';
        }
    },
    watch:{
        selectedRows(newVal){
            if(newVal.length == 0){
                this.bulkSelectShow = false;
            }
        },
    },
    methods: {
        selectable(row, index) {
            // 不是輔站的，才可被選中
            if(row.dual_carrier_type == 2 && row.product != 'RTD') {
                return false;
            }else {
                return true;
            }
        },
        showCollectMessage(row) {
            var vm = this,
                paramsExist = {
                    type: 'enb',
                    operatorCode: operatorCodeGloab
                };

            axios.post(vm.context + '/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
                var data = res.data;

                if(data && data.isExist == 'true') {
                    vm.cllectExisted = true;
                    vm.existedMsgSN = data.serialNumber;
                }else {
                    vm.cllectExisted = false;
                    vm.existedMsgSN = '';
                }
            });

            vm.collectForm.collectInterval = '10';
            vm.collectMessageShow = true;
        },
        sendCollect() {
            var vm = this,
                row = vm.selectedRow || {},
                time = vm.collectForm.collectInterval+':00',
                params = {
                    deviceCode: row.small_cell_code,
                    serialNumber: row.serial_number,
                    type: 'enb',
                    operatorCode: operatorCodeGloab,
                    collectInterval: vm.collectForm.collectInterval
                },
                paramsExist = {
                    type: 'enb',
                    operatorCode: operatorCodeGloab
                };

            axios.post(vm.context + '/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
                var data = res.data;

                if(data && data.isExist == 'true') {
                    vm.$message.error('SN=' + data.serialNumber + vm.local.collecting);
                }else {
                    axios.post(vm.context + '/trace/start.action', stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success == true) {
                            queryGsmVue.startInterval(time);
                            queryGsmVue.queryLatestInfo();
                            vm.collectMessageShow = false;
                            
                            vm.$message.success(vm.local.success);
                        }else {
                            vm.$message.error(data.message);
                        }
                    });
                }
            });
        },
        parseMME(str) {
            if(str) {
                return eval('('+str+')');
            }else {
                return [];
            }
        },
        parsePLMN(row,value,index) {
            var vm = this, stringOrObj = vm.plmnIsJson(value);
            if(stringOrObj == true){
                //对象
                var curValue = JSON.parse(value);
                return Object.keys(curValue).length;
            }
        },
        parseStrPLMN(row,value,index) {
            var vm = this, html = '', stringOrObj = vm.plmnIsJson(value);
            //字符串
            if(stringOrObj == false){
                var list = (value||'').split(','),
                html = list[0];

                if(list.length>1) {
                    html += '<a style="color: blue;" onclick="showPopLayer({event: event, html: &quot;<di style=\'padding-top: 15px;display: block;\'>'+value+'</div>&quot;})">[' + list.length
                            +'<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]</a>';
                }
                return html;
            }
        },
        plmnIsJson(str){
            if(typeof(str) == 'string'){
                try{
                    var curValue = JSON.parse(str);
                    //对象
                    if(typeof curValue == 'object' && curValue){
                        return true;
                    }else{
                        //字符串
                        return false;
                    }
                }catch(e){
                    return false;
                }
            }else{
                return false;
            }
        },
        refreshList() {
            this.$refs.list.refresh();
        },
        loadSuccess(data) {
            this.refresh_cellStatusStatistics();
        },
        showAddOrImport() {
            var vm = this;
            $('.addGsmOrImport-content').html('');
            $('.addGsmOrImport-content').each(function(idx,item){
                if($(item).is(':visible')) {
                    $(item).load(vm.context + '/cell/cpeinfos/toGSMMonitorAddPages.action',function(html) {
                        
                    })
                }
            })
        },
        // 获取所有列
        getAllDefaultCols() {
            var vm = this,
                columns = [
                    {label: vm.local.bscCode, prop: 'serial_number', width: 180},
                    {label: vm.local.bscName, prop: 'host_name', width: 150, formatter: vm.nameFmt},
                    {label: vm.local.rfStatus, prop: 'rf_status', width: 95},
                    {label: vm.local.activeStatus, prop: 'op_state', width: 120},
                    {label: vm.local.ueCount, prop: 'ue_count', width: 80},
                    {label: vm.local.cellIp, prop: 'cell_ip', width: 100},
                    {label: 'MAC', prop: 'mac_address', width: 115},
                ];

            if(supportTopoSite) {
                columns.push({label: vm.local.siteName, prop: 'sub_station_name', width: 110});
            }

            columns.push({label: vm.local.product, prop: 'product', width: 110});
            columns.push({label: vm.local.productName, prop: 'product_name', width: 110});

            columns.push({label: vm.local.module, prop: 'module_type', width: 95});
            columns.push({label: vm.local.software, prop: 'software_version', width: 120});
            columns.push({label: vm.local.groupName, prop: 'group_name', width: 130});
            
            columns.push({label: vm.local.uptime, prop: 'up_time', width: 110});
            
            columns.push({label: vm.local.duration, prop: 'online_duration', width: 110});
            columns.push({label: vm.local.firstTime, prop: 'first_online_time', width: 125});
            columns.push({label: vm.local.lastTime, prop: 'LASTINFORMTIME', width: 220});
            columns.push({label: vm.local.firmware, prop: 'firmware_version', width: 115});

            columns.push({label: "Ipa Unit Id", prop: 'IpaUnitId', width: 120});
            columns.push({label: "Oml Remote Ip", prop: 'OmlRemoteIp', width: 120});
            columns.push({label: "Oml RemoteIp Bak", prop: 'OmlRemoteIpBak', width: 150});
            columns.push({label: "BSC Select", prop: 'BscSelect', width: 100});
            columns.push({label: "BSC Link Status", prop: 'BscLinkStatus', width: 120});
            columns.push({label: "BSC SerialNumber", prop: 'BSCSerialNumber', width: 150});
            columns.push({label: "BTS Num", prop: 'BtsNum', width: 90});
            columns.push({label: vm.local.syncStatus, prop: 'synStatus', width: 160});
            columns.push({label: vm.local.longitude, prop: 'gps_longitude', width: 100});
            columns.push({label: vm.local.latitude, prop: 'gps_latitude', width: 90});
            columns.push({label: vm.local.height, prop: 'gps_height', width: 70});
            columns.push({label: vm.local.satellite, prop: 'gps_satellite_count', width: 75});

            columns.push({label: 'LAC', prop: 'currentLac', width: 70});
            columns.push({label: vm.local.arfcn, prop: 'currentArfcn', width: 70});
            columns.push({label: vm.local.upfreq, prop: 'uplinkFrequency', width: 110});
            columns.push({label: vm.local.downfreq, prop: 'downlinkFrequency', width: 110});

            return columns;
        },
        selectionChange(s) {
            this.selectedRows = s||[];
        },
        closeSyncName() {
            document.body.click();
        },
        clearSelection() {
            this.$refs.list.clearSelection();
        },         
        movecells(){
            var vm = this;
            if(vm.selectedRows.length == 0)return
            vm.groupId = '';
            vm.deviceGroupTip = false;
            vm.deviceGroupMoveShow = true;
        },
        //选择设备选中事件
        groupIdChange(row, old){	    		
            var vm = this;
            if(row) {
                vm.groupId = row.id;	
                vm.deviceGroupTip = false;
            }			
        },
        deviceGroupMoveSubmit(){
            var vm = this, params = {},
                cellCodes = vm.selectedRows.map(function(item){ return item.small_cell_code }).join(',');
            
            params.toGroupId = vm.groupId;
            params.ids = cellCodes;
            //根据被选中的ca列表id进行提示
            if(vm.groupId == '' || vm.groupId == null || vm.groupId == undefined){
                vm.deviceGroupTip = true;
                return false;
            }else{
                vm.deviceGroupTip = false;	    					
            }
            axios.post(vm.context + '/system/deviceGroup/moveCellToGroup.action',stringify(params)).then(function(response){
                let data = response.data;
                if(data.success){
                    vm.$message({
                        message: vm.local.success,
                        type:'success'
                    });	
                    vm.deviceGroupMoveShow = false;
                    vm.clearSelection();
                    vm.refreshList();
                }else{
                    vm.$message({
                        type:'error',
                        message:data.message
                    })
                }
                
            }).catch(function(error){}) 
        },
        syncList() {
            var vm = this,
                codes = vm.selectedRows.map(function(item){ return item.serial_number}),
                rows = vm.$refs.list.getData(),
                cellCodes = '';

            if(vm.selectedRows.length == 0)return

            cellCodes = vm.selectedRows.map(function(item){
                return item.small_cell_code
            }).join(',');
            
            var params = {
                cellCodes : cellCodes
            }
            vm.batchSync = true;
            vm.batchCode = cellCodes;
            vm.openSyncDialog();
            
        },
        closeSyncDialog(){
            var vm = this;
            vm.batchSync = false;
            vm.batchCode = '';
            vm.clearSelection();
            
        },
        rebootList() {
            var vm = this,
                checkedRow = vm.selectedRows,
                cellCodes = '';
            if(vm.selectedRows.length == 0)return
            checkedRow.map(function(item){
                cellCodes += item.small_cell_code + ',';
            });
            
            $.messager.confirm(vm.local.confirm, vm.local.rebootTip, function (r) {
                if (r) {
                    showMsg('prompt_msg',vm.local.sendTip)
                    var param = {
                        cellCodes: cellCodes
                    };
                    $.post(vm.context + "/task/reboot/batchRebootCell.action", param, function (data) {
                        if (data["success"]) {
                            vm.clearSelection();
                        }else{
                            showMsg('error_msg',data["message"]);
                        }
                    }, "json");
                }
            }).addClass("seriousConfirm");
        },
        syncName(code, newName) {
            var vm = this;

            $.post(vm.context + '/cell/cpeinfos/syncCellName.action?smallCellCode='+code, function(data){
                if(data.success){
                    vm.refreshList();
                    vm.closeSyncName();
                }else{
                    showMsg('error_msg',data["message"]);
                }
            },'json'); 
        },
        synchronizeGPS(code) {
            var vm = this;

            $.ajax({
                url: vm.context + '/cell/topo/syncGPSInfo.action',
                type: 'post',
                data: {cell_code: code},
                dataType: 'json',
                success: function(data) {
                    if(data.success){
                        vm.$refs.list.refresh();
                    }else{
                        showMsg('error_msg',data["message"])
                    }
                }
            });
        },
        // table formatters
        connStatusFmt(row, value, index) {

            return connStatusFormatterSyn(value, row, index);
        },
        ueCountsFormatter(rowData, value, rowIndex) {
            var vm = this;
            if(value == 0 ){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>0</a>"; 
            }else if(value == -1 || value == null){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
            }else{
                return "<a style='color:#000000;text-decoration:none;' href='#'>"+value+"</a>";
            }
        },
        ipAddrFmt(row, value, index) {
            if(value) {
                value = '<a href="https://' + value + '" target="_blank" style="color: #4d84ff;">'+value+'</a>';
            }

            return value;
        },
        productFmt(row, value, index) {
            if (this.carrierType == "1") {
                return value;
            }else {
                if(row.have_connected == 2) return '';
                if(value == '--') return '--';

                return value;
            }
        },
        capablityFmt(rowData,value,rowIndex){
            var rowDatas = rowData,
                rowIndexs = rowIndex,
                capablity = rowData.capablity;

            if(capablity == 'enable' && isLWAEnable){
                value = "<span class='cpeLwaKai' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
            }else if(capablity =='disable' && isLWAEnable){
                value = "<span class='cpeLwaWu' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
            }
            return value;
        },
        // table sliders
        closeUeSlide() {
            this.$refs.ueCount.hide();
        },
        //告警数量点击
        alarmToInfo(row){
            var vm = this;
            vm.openSettingPage(row,'alarm');
        },
        openSettingPage(row,page){
            this.selectedRow = row;
            enbPlatform = row.platform_flag;
            enbPlatformType = row.platformType;
            this.settingUrl = this.context + '/cell/cpeinfos/toGSMMonitorSettingPages.action';
            this.$refs.settingPage.showSlide(function(){
                eventBus.$emit("gsm-setting-row",row,page)
            });
        },
        closeSettingPage(){
            this.$refs.settingPage.hide();
        },

        optClick(row, evt) {
            var vm = this,
                rowDatas = row,
                status = rowDatas.connection_status,
                idval = rowDatas.small_cell_code+"",
                flag = rowDatas.platform_flag,
                isCA = rowDatas.ca_flag,
                opState = rowDatas.op_state,
                ip = rowDatas.cell_ip,
                validitySwitch = rowDatas.validity_switch,
                CELL_IDENTITY = rowDatas.CELL_IDENTITY,
                product = rowDatas.product;
                serialNumber = rowDatas.serial_number,
                dualCarrierType = rowDatas.dual_carrier_type,
                collectShow = (is_super_user == 'true') && dualCarrierType != 2;

            vm.selectedRow = row;
            
            var connectStatus = status;
            var deviceType = flag; // intel -- 0 ,gaotong -- 1
            var XinXi = vm.local.info;
            var TongBu = vm.local.sync;
            var SheZhi = vm.local.setting;
            var MeiYouQuanXian = vm.local.notselectall;
            var ChongQi = vm.local.reboot;
            var RiZhi = vm.local.collectLog;
            var MiMaChongZhi = vm.local.modifyPWD;
            var License = vm.local.license;
            var CaoZuo = vm.local.operation;
            var registerSAS = vm.local.registerSAS; 
            var deregisterSAS = vm.local.deregisterSAS;
            var GengDuoCaoZuo = vm.local.moreOpt;
            var WeiHuCaoZuo = vm.local.maintaince;
            var PeiZhiHuiFu = vm.local.restore;
            var RFname = vm.local.rfName;
            var RFon = vm.local.on;
            var RFoff = vm.local.off;
            var YouXiaoQi = vm.local.validDate;
            var LiuLiangXianZhi = vm.local.limitation;
            var FenBuShi = vm.local.dispatch;
            var rfText1,rfText2,
                rfChild = [];
            var rizhidisableflag;
            var rfStatus1,rfStatus2;
            var platformType = rowDatas.platformType,
                dualCarrierType = rowDatas.dual_carrier_type,
                moduleType = rowDatas.module_type;
            
            var showFlagList={
                    CODE_ENB_REBOOT:false,
                    CODE_ENB_RESET_CONFIG:false,
                    CODE_ENB_LOGS:false,
                    CODE_ENB_SYNCHRONIZE:false,
                    CODE_ENB_HALOB_ENABLE:false,
                    CODE_ENB_RF_ENABLE:false,
                    CODE_ENB_ACTIVE:false,
                    CODE_ENB_SAS_RF_ENABLE:false,
                    CODE_ENB_SAS_ENABLE:false,
                    CODE_ENB_LOCK:false,
                    CODE_ENB_CHANGE_PASSWORD:false,
                    CODE_ENB_INFORMATION:false,
                    CODE_ENB_SETTINGS:false,
                    CODE_ENB_EXPIRY_DATE:false,
                    CODE_ENB_EXPIRY_DATE_LOCK:false,
                    CODE_ENB_DISTRIBUTED:false,
                    CODE_ENB_TRAFFIC_LIMITATION:false,
                },
                paramsCode={
                    smallCellCode:idval
                },
                maintenanceShow = false,
                actionShow = false;

            $.ajax({
                type:'POST',
                url: vm.context + '/cell/cpeinfos/getENBOperationItem.action',
                data:paramsCode,
                async:false,
                dataType:'json',
                success:function(data){
                    var operationData = data;
                    if(operationData.Maintenance && operationData.Maintenance.length > 0 ){
                        operationData.Maintenance.map((item)=>{
                            showFlagList[item] = true;
                        });
                    }
                    if(operationData.Actions && operationData.Actions.length > 0 ){
                        operationData.Actions.map((item)=>{
                            showFlagList[item] = true;
                        })
                    }
                    if(operationData.others && operationData.others.length > 0 ){
                        operationData.others.map((item)=>{
                            showFlagList[item] = true;
                        })
                    }
                    
                },
            });

            if(connectStatus != 'Off'){
                rizhidisableflag = false
            }else{
                rizhidisableflag = true;
            }
        
            //配置恢复 判断
            var tongbuDisableFlag;
            if(connectStatus != 'Off'){
                tongbuDisableFlag = false;
            }else{
                tongbuDisableFlag = true;
            }
            
            //判断重启 flag
            var chongQidisableFlag;
            if(connectStatus != 'Off'){
                chongQidisableFlag = false;
            }else{
                chongQidisableFlag = true;
            }
            var isUnusable = false;
            
            var group1 = [
                {row: row, label:TongBu,id:31,cls:'CODE_ENB_SYNCHRONIZE hidden',show:showFlagList.CODE_ENB_SYNCHRONIZE,disable: tongbuDisableFlag,cell_code:idval},
                {row: row, label: vm.local.collectReport,id:50,show: collectShow},
            ]
            var group2 = [
                {row: row, label:ChongQi,id:36,cls:'CODE_ENB_REBOOT hidden',show:showFlagList.CODE_ENB_REBOOT,disable: chongQidisableFlag,cell_code:idval,product:product},
            ]
            var group3 = []
            var group4 = [
                {row: row, label:RiZhi,id:32,cls:'CODE_ENB_LOGS hidden',show:showFlagList.CODE_ENB_LOGS,disable: rizhidisableflag},
            ]
    
            if(rowDatas.have_connected == 2) {// 是否真实可用站
                group1 = [] ;
                group2 = [] ;
                group3 = [] ;
                group4 = [] ;
                isUnusable = true;
            }
            var id4param = idval;
            
            vm.menus = [
                    {row: row,cls:'',show:group1.length>0 && (collectShow || showFlagList.CODE_ENB_SYNCHRONIZE),
                        child: group1, disable: isUnusable
                    },
                    {row: row, cls:'',show:group2.length>0 && showFlagList.CODE_ENB_REBOOT,
                        child: group2, disable: isUnusable
                    },
                    {row: row,cls:'',show: group3.length>0 && group3Flag,
                        child: group3, disable: isUnusable
                    },
                    {row: row,cls:'',show: group4.length>0 && (showFlagList.CODE_ENB_LOGS),
                        child: group4, disable: isUnusable
                    },
                ];
            
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.menu.show(evt);
            })
        },
        menuClick(item) {
            var vm = this,
                row = item.row;

            var rowCode = row.small_cell_code;

            switch(item.id){
            case 1: //信息
                goCellDetailParamInfoWin("enbStatistics", row.small_cell_code, row.connection_status, row.CELL_IDENTITY);
                
                break;
            case 2://设置
                if(row.platformType.indexOf('436Q')>=0 || row.platformType.indexOf('BAIBLQ')>=0 || row.platformType.indexOf('MLQ')>=0){
                    goSettingPanel(row.small_cell_code, row.serial_number, row.connection_status, row.dual_carrier_type);
                }else{
                    halobSwitchFlag = row.halob_flag;
                    jumpToSetting(rowCode, item.label, row.platform_flag);
                }
                break;
            case 3://操作
                
                break;
            case 4://操作
                
                break;
            case 31://同步
                //refreshCell(item.cell_code);
                vm.openSyncDialog(item.cell_code);
                
                break;
            case 32://日志
                vm.gsmConfirmImmediateCollectLogFile(row);
                
                break;
            case 35://sas注册
                registerSas();
                
                break;
            case 36://重启
                cellReboot(item.cell_code,row.product);
                
                break;
            case 37://deregister
                deregisterSAS();
                
                break;
            case 40://Halob开启关闭
                var cellCode = item.cell_code;
                var halobSwitch = item.halob_switch;
                openCloseHalob(cellCode,halobSwitch);//开启关闭Halob操作
                
                break;
            case 41://激活状态
                var cellCode = item.activeStatus;
                var smallcellCode =  item.cell_code;
                if(row.product == 'PM-B4860'){
                    vm.activeCellLialogLoading = true;
                    axios.post(vm.context + '/pm/nxp/getSlotCellInfos.action',stringify({smallCellCode:smallcellCode})).then(function(response){
                        let data = response.data
                        if(data){
                            vm.cellInfosList = vm.activeCellsDataFot(data);
                            vm.activeCellLialogLoading = false;
                        }
                    }).catch(function(error){})
                    vm.showActiveCellDialog = true;
                }else{
                    activeOpStatus(cellCode,smallcellCode);//开启关闭Halob操作
                }
                break;
            case 410://激活状态 - CA cell1小区
                var cellCode = item.activeStatus,
                    smallcellCode =  item.cell_code,
                    cellNumber = item.cellNumber;
                activeOpStatus(cellCode,smallcellCode,cellNumber);//开启关闭Halob操作
                
                break;
            case 411://激活状态 - CA cell2小区
                var cellCode = item.activeStatus,
                    smallcellCode =  item.cell_code,
                    cellNumber = item.cellNumber;
                activeOpStatus(cellCode,smallcellCode,cellNumber);//开启关闭Halob操作
                
                break;
            case 412://激活状态 - CA cell3小区
                var cellCode = item.activeStatus,
                    smallcellCode =  item.cell_code,
                    cellNumber = item.cellNumber;
                activeOpStatus(cellCode,smallcellCode,cellNumber);//开启关闭Halob操作
                
                break;
            case 42://恢复默认配置
                var smallcellCode =  item.cell_code
                configReset(smallcellCode);
                
                break;
            case 43://RF 
                var smallcellCode =  item.cell_code
                setRFStatus(smallcellCode, item.rfStatus);
                
                break;
            case 5:
                var smallcellCode =  item.cell_code
                setEffectPeriod();
                
                break;
            case 431:
                
                break;
            case 432://RF 关闭
                var smallcellCode =  item.cell_code
                setRFStatus(smallcellCode,item.rfStatus,item.cellNumber)
                
                break; 
            case 48://autoOrForceRF
                var serialNumber =  row.serial_number;
                setForceRFStatus(serialNumber, item.rfStatus)
                
                break; 
            case 'limit': // Limitation
                setLimitation(rowCode);
                
                break; 
            case 6://分布式基站
                distributeSn("enbDistribute",row.small_cell_code);
                
                break;
            
            case 50://收集报文
                vm.showCollectMessage(row);
                
                break;
            }
        },
        handerClose() {
            this.$refs.menu.hide();
        },
        closeInfo() {
            this.$refs.info.hide();
        },
        activeCellsDataFot(data){  // 板卡小区格式化
            var cellInfosList = [
                {label:'1',cellItem:[]},
                {label:'2',cellItem:[]},
                {label:'3',cellItem:[]},
                {label:'4',cellItem:[]},
            ];
            data.map((item,index)=>{
                item.oldStatus = item.activeStatus;
                if(item.cardId == '1'){
                    cellInfosList[0].cellItem.push(item)
                }else if(item.cardId == '2'){
                    cellInfosList[1].cellItem.push(item)
                }else if(item.cardId == '3'){
                    cellInfosList[2].cellItem.push(item)
                }else if(item.cardId == '4'){
                    cellInfosList[3].cellItem.push(item)
                }
            });
            cellInfosList.map((item,index,arr)=>{
                if(item.cellItem.length>0){
                    item.cellItem = item.cellItem.sort((a,b)=>{
                        return parseInt(a.cellIndex) - parseInt(b.cellIndex)
                    })
                }
                
            })
            return cellInfosList
        },
        activeCellSubmit(){ // PM-B4860基站 激活/取消激活
            var vm = this,
                url = vm.context + '/cell/cpeinfos/cellModifyActiveStatus.action',
                params = {
                    small_cell_code: vm.selectedRow.small_cell_code, // vm.selectedRow.small_cell_code
                    cellNumber:''
                },
                cellNumber = [];

            vm.cellInfosList.map((item,index)=>{
                if(item.cellItem.length>0){
                    item.cellItem.map((items,idx)=>{
                        cellNumber.push({cellIndex:items.cellIndex,status:items.activeStatus})
                    })
                }
            })
            cellNumber = JSON.stringify(cellNumber);
            params.cellNumber = cellNumber;
            axios.post(url,stringify(params)).then(function(response){
                let data = response.data
                if(data["success"]){
                    vm.$message.success(vm.local.success)
                    vm.refreshList();
                        vm.showActiveCellDialog = false;
                }else{
                    vm.$message.error(data.message) //错误提示信息 
                }
            }).catch(function(error){})
        },
        closeActiveCellDialog(){ // 关闭 PM-B4860基站 激活/取消激活弹窗
            var vm = this;
            vm.cellInfosList = [];
            vm.showActiveCellDialog = false;
        },
        cellActivePopoverShow(row){ // 激活信息Popover 打开事件
            var vm = this;

            vm.cellActivePopoverLoading = true;
            axios.post(vm.context + '/pm/nxp/getSlotCellInfos.action',stringify({smallCellCode:row.small_cell_code})).then(function(response){
                let data = response.data
                if(data){
                    vm.activeCellsDataList = vm.activeCellsDataFot(data);
                    vm.cellActivePopoverLoading = false;
                }
            }).catch(function(error){})
        },
        cellActivePopoverHide(){ // 激活信息Popover 关闭事件
            var vm = this;
            vm.activeCellsDataList = [];
        },
        // 打开已选弹窗
        openBulkSelectTable(){
            var vm = this;
            vm.bulkSelectShow = true
        },
        // 关闭已选弹窗
        closeBulkSelectTable(){
            var vm = this;
            vm.bulkSelectShow = false;
        },
        // 设备已选表格 清空事件
        clearBulkSelected(){
            var vm = this;

            vm.$refs["list"].clearSelection();
        },
        // 设备已选表格 单个删除事件
        delBulkSelected(rows){
            var vm = this,
                tabs = 'list',
                rowKey = 'serial_number';
            vm.selectedRows = vm.selectedRows.filter((items)=>{
                return items[rowKey] != rows[rowKey]
            });
            var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
                irow= selection.filter((items)=>{
                    return items[rowKey] == rows[rowKey]
                })[0];
            vm.$refs[tabs].toggleRowSelection(irow,false);
            var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
            vm.$refs[tabs].ckList.splice(idx,1);
        },
        // 判断小区状态 RF状态 显示  1 全部在线 2 部分在线 3 全部不在线
        judgeActiveStatusFat(value){
            var status = '1',
                states = (value+'').split(',');
            if((states.indexOf('0') >= 0  && states.indexOf('1') >= 0) || (states.indexOf('off') >= 0  && states.indexOf('on') >= 0)){
                status = '2'
            }else if((states.indexOf('0') >= 0  && states.indexOf('1') < 0) || (states.indexOf('off') >= 0  && states.indexOf('on') < 0)){
                status = '3'
            }else if((states.indexOf('0') < 0  && states.indexOf('1') >= 0) || (states.indexOf('off') < 0  && states.indexOf('on') >= 0)){
                status = '1'
            }
            return status
        },
        // 判断小区装填 RF状态  在线数 与 总数
        judgeActiveNumOrAllNumFat(value,type){
            var activeNum = [],
                allNum = [],
                states = (value+'').split(','),
                val = 0;
            states.map((item,index)=>{
                if(item == '1' || item == 'on'){
                    activeNum.push(item)
                }
                allNum.push(item)
            })

            if(type == 'active'){
                val = activeNum.length
            }else{
                val = allNum.length
            }
            return val
        },
        // 生成 小区状态 RF状态 集合
        parseCellAndRfStatus(value){
            var states = (value+'').split(',');
            return states
        },
        init(){
            var vm = this;

            vm.operatorCtableDefaultCheckList.push(operator_code);
            
            this.refresh_cellStatusStatistics();
        },
        openSyncDialog() {
            var vm = this;
            vm.openSyncDialogShow = true;
            vm.isSyncloading = true;

            setTimeout(function() {
                $('#gsm_sync-content').html('');
                $('#gsm_sync-content').each(function(idx,item){
                    if($(item).is(':visible')) {
                        $(item).load(vm.context + '/cell/cpeinfos/toGSMMonitorSyncParamsPages.action',function(html) {
                            vm.isSyncloading = false;
                        })
                    }
                })
            },200);
        },
        showGsmExport(){
            var vm = this;
            
            vm.$nextTick(function(){
                var operator_code = operator_code;
                vm.$refs.operatorCtable.refresh();
            })
        },
        operatorCtableHandleChange(s){
            var vm = this;
            vm.operatorCtableCheckList = s;
        },
        exportCSV(type){
            var vm = this,
                params = {
                    TimeZone: timeZone,
                    operator_codes: operator_code,
                    content: '',
                    sort: gsmvm.$refs.list.sortName,
                    order: gsmvm.$refs.list.sortOrder,
                };
            
            params.content = 'connection_status,alarm,' + vm.showColums.join(',');

            var queryParams = gsmvm.queryParams;

            Object.assign(params,queryParams);
            if(type == 'monitor'){
                downLoadFileByAxios(vm.context + '/cell/cpeinfos/exportGSMInfosToExcel.action', params);
            }else if(type == 'license'){
                params.isGSM = 1;
                downLoadFileByAxios(vm.context + '/cell/cpeinfos/exportEnodebLicenseInfos.action', params);
            }
            document.body.click();
        },
        comfirmExport() {
            var vm = this,
                params = {
                    TimeZone: timeZone,
                    operator_codes: '',
                    content: '',
                    sort: gsmvm.$refs.list.sortName,
                    order: gsmvm.$refs.list.sortOrder,
                };
            
            params.content = 'connection_status,alarm,' + vm.showColums.join(',');

            if(vm.operatorShow) {// 云环境且为amdin才可以选择运营商
                if(vm.operatorCtableCheckList && vm.operatorCtableCheckList.length > 0) {
                    params['operator_codes'] = vm.operatorCtableCheckList.map(function(row){
                        return row.operator_code;
                    }).join(',');
                }
            }else {// 运营商和普通用户
                params['operator_codes'] = operator_code;
            }

            var queryParams = gsmvm.queryParams;

            Object.assign(params,queryParams);
            exportByForm(vm.context + '/cell/cpeinfos/exportGSMInfosToExcel.action', params);
            setTimeout(function(){
                if(vm.isExportDeviceLicenseInfo){
                    params.isGSM = 1;
                    exportByForm(vm.context + '/cell/cpeinfos/exportEnodebLicenseInfos.action', params);
                    vm.isExportDeviceLicenseInfo = false;
                }
            },1000);
            document.body.click();
        },
        refresh_cellStatusStatistics(cb) {
            var params = {},dualStatus = '';
            
            $.extend(params, this.queryParams);

            params["switch_status"] = dualStatus;
            params["isDual"] = true;
            params["isMonitor"] = true;
            params["isGSM"] = '1';
            
            $.post(this.context + "/cell/cpeinfos/getCellStatusStatistics.action", params, function(data) {
                if(!data["connection_status"]){
                    connectionStatus = "0/0";
                    connectionStatusRef = "0/0"
                }else{
                    connectionStatus = data["connection_status"];
                    connectionStatusRef = data["connection_status_ref"];
                }
                $('#gsm_online_count_rate').text("( "+connectionStatus+" )");
                
                if(cb && typeof cb == 'function') {
                    cb();
                }
            }, "json");
        },
        gsmConfirmImmediateCollectLogFile(row) {
            var vm = this;
            var selCell = vm.selectedRow,
                cellCode = selCell.small_cell_code,
                serial_number = selCell.serial_number,
                param = {
                    start_time: 'undefined',
                    end_time: 'undefined',
                    execute_type: 'Immediately',
                    reportPeriod: '',
                    isReboot: 'false',
                    serial_number: serial_number, 				    				
                    timeZone: timeZone, 
                    device_type: "eNB",
                    device_code: cellCode,
                    logType: 'deviceLog'
                };
            
            $.post(vm.context + "/cell/collect/goImmediateCollectLogFile.action", param, function (data) {
                if (data["success"]) {
                    showMsg('success_msg', vm.local.logCollecting)
                } else {
                    showMsg('error_msg',data['message']);
                }
            }, "json");
        },
        getGsmGPSSignalData(rowData) {
            var vm = this,
                codes = rowData.small_cell_code,
                url = vm.context + "/cell/param/getBTSSatellitesDataList.action?timeZone="+timeZone+'&smallCellCode='+codes+'&rd='+Math.random();
            
            enbvm.satelliteURL = url;
            enbvm.sateloading = true;
            enbvm.$refs.satelite.showSlide(function(){
                setTimeout(function(){
                    enbvm.sateloading = false;
                },1000);
            });
        },
        closeSatellite() {
            enbvm.$refs.satelite.hide();
        },
        // 同步状态格式化
        syncStatusFmt(rowData, value, rowIndex){
            if (value == null) {
                return null;
            } else if (value == "--" ){
                return "--";
            } else if (value == ("GPS " + this.local.inSync)) {
                val ="GPS "+ this.local.inSync;
                value = "<span>"+(val)+"</span>"
            }else if (value == ("1588 " + this.local.inSync)) {
                val ="1588 "+ this.local.inSync;
                value = "<span>"+(val)+"</span>"
            }else if (value == ("REM " + this.local.inSync)) {
                val ="REM "+ this.local.inSync;
                value = "<span>"+(val)+"</span>"
            }else if (value == "GPS "+ this.local.syncSuccess ) {
                val ="GPS "+ this.local.syncSuccess;
                value = "<span>"+(val)+"</span>"
            }else if (value == "1588 "+this.local.syncSuccess ) {	
                val = this.local.syncSuccess;
                val = "1588 " + val;
                value = "<span>"+(val)+"</span>"
            }else if (value == "REM "+this.local.syncSuccess) {	
                val = this.local.syncSuccess;
                val = "REM " + val;
                value = "<span>"+(val)+"</span>"
            }else if (value == this.local.notSync) {	
                val = this.local.notSync;
                value = "<span class='offStatusCls'>"+(val)+"</span>"
            }
            return '<div style="min-width: 130px;diplay: line-block;">'+value+'</div>';
        }
    },
    mounted() {
        // 排序
        var vm = this,
            sortCodes = enbShowCols.split(','),
            sortList = vm.columns;
        vm.init();
        
        eventBus.$on('cancel-enb-setting',vm.closeSettingPage);

        window.gsmvm = vm;
    }
})
