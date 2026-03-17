<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>GSM Monitor</title>
    <script type="text/javascript" src="${ctx}/js/element/Sortable.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/element/vuedraggable.umd.min.js?_=${omc_ver}"></script>

    <style>
		#gsmTableHeadQuery .headQueryBox .queryGroup .el-input__inner{
            width: 360px;
        }
        #gsmMonitorPage .exportDrop{
            width:30px;
            height:30px;
            position:absolute;
            right:20px;
            top:10px;
        }
        #gsmMonitorPage .remarkHeaderCls ,#gsmMonitorPage .remarkHeaderCls div{
            padding-left: unset;
            line-height: unset;
        }
        #gsmMonitorPage .remarkHeaderCls .el-input__suffix{
            line-height: 30px;
        }
        #gsmMonitorPage .remarkHeaderCls .el-input--suffix .el-input__inner{
            padding-right: 50px;
            box-sizing: border-box;
        }
        #gsmMonitorPage .remarkHeaderCls .el-icon::before{
            font-size: 16px;
            color: #7A7992;
        }
    </style>
</head>
<body>
    <div id="gsmMonitorPage" class="monitor-ctner" style="overflow: auto;background-color: #fff;display: flex; flex-direction: column; height: 100%;">
        <div style="height: 100%; flex:auto; overflow: auto;">
            <vxe-grid ref="list" height="100%"
                id="gsmTableHomeCellList"
                border
                size="mini"
                show-overflow="tooltip"
                show-header-overflow="tooltip"
                :row-config="{keyField: 'serial_number', isHover: true}"
                :data="tableData"
                :columns="tableColumns"
                :column-config="{resizable: true}"
                :checkbox-config="{checkRowKeys: checkedRowKeys, reserve: true, checkMethod: checkSelectable}"
                :scroll-y="{enabled: true, gt: 50, oSize: 10}"
                :scroll-x="{enabled: true, gt: 20}"
                :sort-config="{remote: true, trigger: 'cell'}"
                :show-overflow="true"
                :stripe="true"
                @checkbox-change="handleSelectionChange"
                @checkbox-all="handleSelectionChange"
                @sort-change="handleSortChange">
                <template #toolbar>
                    <div class="toolbarHeadBtnBoxCls">
                        <div id="addGsmDeviceOrImport" @click="showAddOrImport">
                            <!-- <el-popover trigger="click" placement="bottom-end" popper-class="monitorBtnPopperCls">
                                <div class="el-card__header">
                                    <%=rb.getString("TianJia")%>
                                    <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                                </div>
                                <div class="addGsmOrImport-content" style="width: 700px; max-height: 500px;overflow: auto;"></div>
                                <div slot="reference" class="newIconBoxCls-bt CODE_ENB_DEVICE_REGISTER hidden" style="right:56px;top:5px;" tip="<%=rb.getString("TianJia")%>">
                                    <span class="el-icon-plus el-icon"></span>
                                </div>	
                            </el-popover> -->
                            <div  v-if="operatorShow" @click="showGsmExport">
                                <el-popover trigger="click" placement="bottom-end" popper-class="monitorBtnPopperCls">
                                    <div class="el-card__header">
                                        <%=rb.getString("DaoChu")%>
                                        <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                                    </div>
                                    <div style="width: 920px; max-height: 500px;overflow: auto;">
                                        <el-ctable ref="operatorCtable" :url="operatorCtableUrl" row-key="operator_code" style="margin: 0px 20px 20px 20px;border: 1px solid #f3f3f3;" height="200" :default-checked="operatorCtableDefaultCheckList" @selection-change="operatorCtableHandleChange">
                                            <el-table-column prop="operator_code" type="selection" :reserve-selection="true"></el-table-column>
                                            <el-table-column prop="operator_name" label="<%=rb.getString("YunYingShangMingCheng")%>"></el-table-column>
                                        </el-ctable>
                                        <div style="margin-left: 20px;">
                                            <div style="font-size: 14px;color:#606266;font-weight:bold;"><%=rb.getString("LicenseXinXi")%></div>
                                            <el-checkbox v-model="isExportDeviceLicenseInfo" style="margin: 10px 20px;"><%=rb.getString("DaoChuLicenseXinXi")%></el-checkbox> 
                                        </div>
                                        <hr style="border: none;border-bottom: 1px solid #e4e7ec;margin: 10px 0;">
                                        <div style="margin-left: 20px;">
                                            <div style="font-size: 14px;color:#606266;font-weight:bold;"><%=rb.getString("DaoChuGeShi")%></div>
                                            <el-radio-group v-model="gsmExportFormatType">
                                                <el-radio style="margin: 10px 20px;" label="csv"><%=rb.getString("CSVGeShi")%></el-radio> 
                                                <el-radio style="margin: 10px 20px;" label="xlsx"><%=rb.getString("XLSXGeShi")%></el-radio> 
                                            </el-radio-group>
                                        </div>
                                        <div style="padding: 20px;">
                                            <el-button type="primary" @click="comfirmExport"><%=rb.getString("QueDing")%></el-button>
                                            <el-button @click="close"><%=rb.getString("QuXiao")%></el-button>
                                            <span style="color: 999;font-weight: normal;margin-left: 20px;"><i class="el-icon el-icon-circle-info gray-color"></i> <%=rb.getString("DengDaiTiShi")%></span>
                                        </div>
                                    </div>
                                    <div slot="reference" class="newIconBoxCls-bt" style="right:20px;top:5px;" tip="<%=rb.getString("DaoChu")%>">
                                        <span class="el-icon-operation-export el-icon"></span>
                                    </div>	
                                </el-popover>
                            </div>
                            <el-dropdown v-if="!operatorShow" @command="exportCSV" trigger="click" class="exportDrop">
                                <div class="newIconBoxCls-bt" style="right:0px;top:-5px;" tip="<%=rb.getString("DaoChu")%>">
                                    <span class="el-icon-operation-export el-icon"></span>
                                </div>	
                                <el-dropdown-menu slot="dropdown">
                                    <el-dropdown-item command="monitor_xlsx"><%=rb.getString("DaoChuJianKongShuJu")%> xlsx</el-dropdown-item>
                                    <el-dropdown-item command="monitor_csv"><%=rb.getString("DaoChuJianKongShuJu")%> csv</el-dropdown-item>
                                    <el-dropdown-item command="license"><%=rb.getString("DaoChuLicenseXinXi")%></el-dropdown-item>
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
                                        <span><%=rb.getString("YiXuan")%></span>
                                        <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                    </div>
                                    <div class="selectBoxMain">
                                        <div class="tableInfoCls">
                                            <div class="tableInfoHeader">
                                                <div><%=rb.getString("Title_SheBeiBianMa")%></div>
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
                            <span><%=rb.getString("YiDongDaoSheBeiZu")%></span>
                        </div>
                        <div :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="syncList">
                            <span class="el-icon el-icon-operation-synchronize"></span>
                            <span><%=rb.getString("TongBu")%></span>
                        </div>
                        <div :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="rebootList">
                            <span class="el-icon el-icon-operation-reboot"></span>
                            <span><%=rb.getString("ChongQi")%></span>
                        </div>
                    </div>
                    <div id="gsmtoolbar_tableHomeCellList" style="position:relative;"></div>
                </template>
                
                <!-- 序号列 - 根据分页动态计算 -->
                <template #row_index="{rowIndex}">
                    <span>{{ (pageParams.pageNum - 1) * pageParams.pageSize + rowIndex + 1 }}</span>
                </template>
                
                <!-- 操作列 -->
                <template #enb_operation="{row}">
                    <div class="el-icon el-icon-circle-setting1" @click="openSettingPage(row)" style="font-size:19px;"></div>
                    <div v-if="isMonitorWritable" class="el-icon el-icon-operation-more-circle" @click="optClick(row, $event)" v-clickoutside="handerClose"></div>
                </template>

                <!-- 连接状态列 -->
                <template #connection_status="{row, rowIndex}">
                    <div v-html="connStatusFmt(row, row['connection_status'], rowIndex)"></div>
                </template>

                <!-- 告警数量列 -->
                <template #alarm_count="{row}">
                    <div style="text-align: center;">
                        <span v-if="row.alarm_serverity == '31001'" class="alarmCritical alarmListSty" @click="alarmToInfo(row, $event)">{{row.alarm_count}}</span>
                        <span v-else-if="row.alarm_serverity == '31002'" class="alarmMajor alarmListSty" @click="alarmToInfo(row, $event)">{{row.alarm_count}}</span>
                        <span v-else-if="row.alarm_serverity == '31003'" class="alarmMinor alarmListSty" @click="alarmToInfo(row, $event)">{{row.alarm_count}}</span>
                        <span v-else-if="row.alarm_serverity == '31004'" class="alarmWarning alarmListSty" @click="alarmToInfo(row, $event)">{{row.alarm_count}}</span>
                        <span v-else>0</span>
                    </div>
                </template>

                <!-- BSC名称列 -->
                <template #host_name="{row}">
                    <div v-if="row.device_name_tip === '0'">{{row.host_name}}</div>
                    <div v-if="row.device_name_tip !== '0'" class="cellNameClass">
                        <el-popover trigger="click">
                            <div slot="reference">
                                <span class="el-icon el-icon-circle-warning"></span> 
                                {{row.host_name}}
                            </div>
                            <div style='padding: 30px 20px 20px; position:relative;'>
                                <div> <span class="el-icon el-icon-close" @click="closeSyncName" style="top: 10px; position:absolute;"></span>
                                <div style='display: flex;'><span style='color: #333; font-size: 14px;'><%=rb.getString("JiZhanCeMingCheng")%> : </span> <span style='font-size: 14px;margin-left: 6px;color: #333;'>{{row.report_host_name}}</span></div>
                                <div style='font-size: 12px; color: #333; margin-top: 6px;'><%=rb.getString("ShiFouTongBuMingChengDaoOMC")%><div>
                                <div style="margin-top: 10px; text-align: right;">
                                    <span class="el-button--primary el-button" @click="syncName(row.small_cell_code, row.report_host_name)">
                                        <%=rb.getString("QueDing")%>
                                    </span>
                                    <span class="white el-button" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
                                </div>
                            </div>
                        </el-popover>
                    </div>
                </template>

                <!-- RF状态列 -->
                <template #rf_status="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-if="row.product == 'BTS'" style="display: flex;align-items: center;">
                        <div v-if="['','NULL','null',null,undefined].includes(row.rf_status)"></div>
                        <div v-if="row.rf_status == '--'">--</div>
                        <div v-if="!['','NULL','null',null,undefined].includes(row.rf_status) && row.rf_status != '--'">
                            <div v-if="row.rf_status == 'on' && parseCellAndRfStatus(row.rf_status).length == 1" class='iconFlexCls'>
                                <span style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
                            </div>
                            <div v-if="row.rf_status == 'off' && parseCellAndRfStatus(row.rf_status).length == 1" class='iconFlexCls'>
                                <span class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
                            </div>
                            <div v-if="parseCellAndRfStatus(row.rf_status).length > 1" style="display: flex;">
                                <span v-if="judgeActiveStatusFat(row.rf_status) == '1'" style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
                                <span v-if="judgeActiveStatusFat(row.rf_status) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
                                <span v-if="judgeActiveStatusFat(row.rf_status) == '3'" class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
                                <el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
                                    <span style="color:#4d84ff;" slot="reference" v-if="parseCellAndRfStatus(row.rf_status).length>1">
                                        [ {{judgeActiveNumOrAllNumFat(row.rf_status,'active')}}/{{judgeActiveNumOrAllNumFat(row.rf_status,'all')}} ]
                                    </span>
                                    <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
                                        <div v-for="(item,index) in parseCellAndRfStatus(row.rf_status)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
                                            Cell {{index+1}}:
                                            <span v-if="item == 'on'" class="onStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Kai")%></span>
                                            <span v-if="item == 'off'" class="offStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Guan")%></span>
                                        </div>
                                    </div>
                                </el-popover>
                            </div>
                        </div>
                    </div>
                </template>

                <!-- 是否激活列 -->
                <template #op_state="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-if="row.product == 'BTS'" style="display: flex;align-items: center;">
                        <div v-if="row.product != 'PM-B4860'">
                            <div v-if="row.op_state == '1' && !['Intel_CR_CA','Intel_CR_TC','MLN_CA'].includes(row.platformType)" class='iconFlexCls'>
                                <span style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                            </div>
                            <div v-if="row.op_state == '0' && !['Intel_CR_CA','Intel_CR_TC','MLN_CA'].includes(row.platformType)" class='iconFlexCls'>
                                <span class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
                            </div>
                            <div v-if="['Intel_CR_CA','Intel_CR_TC','MLN_CA'].includes(row.platformType)" style="display: flex;">
                                <span v-if="judgeActiveStatusFat(row.op_state) == '1'" style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                <span v-if="judgeActiveStatusFat(row.op_state) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                <span v-if="judgeActiveStatusFat(row.op_state) == '3'" class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                <el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
                                    <span style="color:#4d84ff;" slot="reference">
                                        [ {{judgeActiveNumOrAllNumFat(row.op_state,'active')}}/{{judgeActiveNumOrAllNumFat(row.op_state,'all')}} ]
                                    </span>
                                    <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
                                        <div v-for="(item,index) in parseCellAndRfStatus(row.op_state)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
                                            Cell {{index+1}}:
                                            <span v-if="item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("JiHuo")%></span>
                                            <span v-if="item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                        </div>
                                    </div>
                                </el-popover>
                            </div>
                            <div v-if="['QA_436Q_CA'].includes(row.platformType)" style="display: flex;">
                                <span v-if="judgeActiveStatusFat(row.op_state) == '1'" style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                <span v-if="judgeActiveStatusFat(row.op_state) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                <span v-if="judgeActiveStatusFat(row.op_state) == '3'" class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                <el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
                                    <span style="color:#4d84ff;" slot="reference">
                                        [ {{judgeActiveNumOrAllNumFat(row.op_state,'active')}}/{{judgeActiveNumOrAllNumFat(row.op_state,'all')}} ]
                                    </span>
                                    <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
                                        <div v-for="(item,index) in parseCellAndRfStatus(row.op_state)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
                                            Cell {{index+1}}:
                                            <span v-if="item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("JiHuo")%></span>
                                            <span v-if="item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                        </div>
                                    </div>
                                </el-popover>
                            </div>
                        </div>
                        <div v-if="row.product == 'PM-B4860'">
                            <div class='iconFlexCls'>
                                <span v-if="row.op_state == '0'" class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
                                <span v-if="row.op_state != '0'" class='offOrOnStatusCls' style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
                                <el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='640' @show="cellActivePopoverShow(row)" @hide="cellActivePopoverHide">
                                    <span style="color:#4d84ff;" slot="reference">
                                        [ {{row.op_state}}/{{row.op_state_cell2}} ]
                                    </span>
                                    <div style="padding:10px;border-top:1px solid #E9E9E9;min-height:120px;" v-loading="cellActivePopoverLoading">
                                        <div v-for="item in activeCellsDataList" v-show='item.cellItem.length>0' style="height:40px;display:flex;align-items: center;margin-bottom:10px;">
                                            <div style="width:110px;">{{item.label}}（<%= rb.getString("BanKaID")%>）</div>
                                            <div v-for="(items,index) in item.cellItem" style="padding-left:10px;height:40px;width:160px;display:flex;align-items: center;justify-content: center;border:1px solid #E9E9E9;">
                                                <div v-if="items.activeStatus == '1'">
                                                    <span style="margin-right: 5px;">{{items.cellIndex}}</span>
                                                    <span class="onStatusBoxCls"><%= rb.getString("JiHuo")%></span>
                                                </div>
                                                <div v-if="items.activeStatus == '0'">
                                                    <span style="margin-right: 5px;">{{items.cellIndex}}</span>
                                                    <span class="offStatusBoxCls"><%= rb.getString("QuJiHuo")%></span>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </el-popover>
                            </div>
                        </div>
                        <el-tooltip v-if="row.delay_avaliable == '0' || row.delay_avaliable == '1'" content="<%=rb.getString("licenseGuoQi")%>">
                            <span class="warnTips el-icon el-icon-sas-warning" style='margin-left: 5px;'></span>
                        </el-tooltip>
                    </div>
                </template>

                <!-- 产品类型标识列 -->
                <template #product="{row, rowIndex}">
                    <div v-html="productFmt(row, row.product, rowIndex)"></div>
                </template>

                <!-- 设备型号名列 -->
                <template #module_type="{row, rowIndex}">
                    <div v-html="capablityFmt(row, row.module_type, rowIndex)"></div>
                </template>

                <!-- IP地址列 -->
                <template #cell_ip="{row, rowIndex}">
                    <div v-html="ipAddrFmt(row, row.cell_ip, rowIndex)"></div>
                </template>

                <!-- UE数列 -->
                <template #ue_count="{row, rowIndex}">
                    <div v-html="ueCountsFormatter(row, row.ue_count, rowIndex)"></div>
                </template>

                <!-- Ipa Unit Id列 -->
                <template #ipa_unit_id="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>{{row.IpaUnitId}}</div>
                </template>

                <!-- Oml Remote Ip列 -->
                <template #oml_remote_ip="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>{{row.OmlRemoteIp}}</div>
                </template>

                <!-- Oml Remote Ip Bak列 -->
                <template #oml_remote_ip_bak="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>{{row.OmlRemoteIpBak}}</div>
                </template>

                <!-- BSC Select列 -->
                <template #bsc_select="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>
                        <span v-if="row.BscSelect == '0'"><%=rb.getString("Zhu")%></span>
                        <span v-else-if="row.BscSelect == '1'"><%=rb.getString("Bei")%></span>
                        <span v-else>{{row.BscSelect}}</span>
                    </div>
                </template>

                <!-- BSC连接状态列 -->
                <template #bsc_link_status="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>
                        <span v-if="row.BscLinkStatus == '0'"><%=rb.getString("MMEWeiLianJie")%></span>
                        <span v-else-if="row.BscLinkStatus == '1'"><%=rb.getString("MMEYiLianJie")%></span>
                        <span v-else>{{row.BscLinkStatus}}</span>
                    </div>
                </template>

                <!-- 所属BSC编码列 -->
                <template #bsc_serial_number="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>{{row.BSCSerialNumber}}</div>
                </template>

                <!-- BTS数列 -->
                <template #bts_num="{row}">
                    <div v-if="row.product == 'BSC'">{{row.BtsNum}}</div>
                    <div v-else>--</div>
                </template>

                <!-- 同步状态列 -->
                <template #syn_status="{row, rowIndex}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else v-html="syncStatusFmt(row, row.synStatus, rowIndex)"></div>
                </template>

                <!-- GPS卫星数列 -->
                <template #gps_satellite_count="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-if="row.product == 'BTS'">
                        <div v-if='row.hasSatelliteDetail == "true"'>
                            <a style='color:#1DA3FC;text-decoration:underline' href='#' @click='getGsmGPSSignalData(row)'>{{row.gps_satellite_count}}</a>
                        </div>
                        <div v-if='row.hasSatelliteDetail != "true"'>{{row.gps_satellite_count}}</div>
                    </div>
                </template>

                <!-- GPS经度列 -->
                <template #gps_longitude="{row, rowIndex}">
                    <div v-if="row.gps_modify_flag != '1'">{{row.gps_longitude}}</div>
                    <div v-else>
                        <div v-if="row.gps_longitude === undefined" :key="rowIndex">
                            {{row.modify_longitude == undefined? row.gps_longitude : row.modify_longitude}}
                        </div>
                        <div v-if="row.gps_longitude !== undefined" class="cellNameClass" :key="rowIndex">
                            <el-popover trigger="click">
                                <div slot="reference">
                                    <span class="el-icon el-icon-circle-warning"></span> 
                                    {{row.modify_longitude}}
                                </div>
                                <div class="gpsSyncPopoverContent">
                                    <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                    <div style="margin-bottom: 10px;">
                                        <%=rb.getString("JingDu")%>: {{row.gps_longitude}}&nbsp;&nbsp;
                                        <%=rb.getString("WeiDu")%>: {{row.gps_latitude}}&nbsp;&nbsp;
                                        <%=rb.getString("GaoDu")%>: {{row.gps_height}}
                                    </div>
                                    <div style="margin-bottom: 10px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                    <div style="text-align: right;margin: 0px;">
                                        <el-button type="primary" size="mini" @click="synchronizeGPS(row.small_cell_code)">
                                            <%=rb.getString("QueDing")%>
                                        </el-button>
                                        <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                    </div>
                                </div>
                            </el-popover>
                        </div>
                    </div>
                </template>

                <!-- GPS纬度列 -->
                <template #gps_latitude="{row, rowIndex}">
                    <div v-if="row.gps_modify_flag != '1'">{{row.gps_latitude}}</div>
                    <div v-else>
                        <div v-if="row.gps_latitude === undefined" :key="rowIndex">
                            {{row.modify_latitude == undefined? row.gps_latitude : row.modify_latitude}}
                        </div>
                        <div v-if="row.gps_latitude !== undefined" class="cellNameClass" :key="rowIndex">
                            <el-popover trigger="click">
                                <div slot="reference">
                                    <span class="el-icon el-icon-circle-warning"></span> 
                                    {{row.modify_latitude}}
                                </div>
                                <div class="gpsSyncPopoverContent">
                                    <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                    <div style="margin-bottom: 10px;">
                                        <%=rb.getString("JingDu")%>: {{row.gps_longitude}}&nbsp;&nbsp;
                                        <%=rb.getString("WeiDu")%>: {{row.gps_latitude}}&nbsp;&nbsp;
                                        <%=rb.getString("GaoDu")%>: {{row.gps_height}}
                                    </div>
                                    <div style="margin-bottom: 20px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                    <div style="text-align: right;margin: 0px;">
                                        <el-button type="primary" size="mini" @click="synchronizeGPS(row.small_cell_code)">
                                            <%=rb.getString("QueDing")%>
                                        </el-button>
                                        <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                    </div>
                                </div>
                            </el-popover>
                        </div>
                    </div>
                </template>

                <!-- GPS高度列 -->
                <template #gps_height="{row, rowIndex}">
                    <div v-if="row.gps_modify_flag != '1'">{{row.gps_height}}</div>
                    <div v-else>
                        <div v-if="row.gps_height === undefined" :key="rowIndex">
                            {{row.modify_height == undefined? row.gps_height : row.modify_height}}
                        </div>
                        <div v-if="row.gps_height !== undefined" class="cellNameClass" :key="rowIndex">
                            <el-popover trigger="click">
                                <div slot="reference">
                                    <span class="el-icon el-icon-circle-warning"></span> 
                                    {{row.modify_height}}
                                </div>
                                <div class="gpsSyncPopoverContent">
                                    <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                    <div style="margin-bottom: 10px;">
                                        <%=rb.getString("JingDu")%>: {{row.gps_longitude}}&nbsp;&nbsp;
                                        <%=rb.getString("WeiDu")%>: {{row.gps_latitude}}&nbsp;&nbsp;
                                        <%=rb.getString("GaoDu")%>: {{row.gps_height}}
                                    </div>
                                    <div style="margin-bottom: 20px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                    <div style="text-align: right;margin: 0px;">
                                        <el-button type="primary" size="mini" @click="synchronizeGPS(row.small_cell_code)">
                                            <%=rb.getString("QueDing")%>
                                        </el-button>
                                        <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                    </div>
                                </div>
                            </el-popover>
                        </div>
                    </div>
                </template>

                <!-- LAC列 -->
                <template #current_lac="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>{{row.currentLac}}</div>
                </template>

                <!-- 频点列 -->
                <template #current_arfcn="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>{{row.currentArfcn}}</div>
                </template>

                <!-- 上行频率列 -->
                <template #uplink_frequency="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>
                        {{row.uplinkFrequency}}
                        <span v-if="row.uplinkFrequency">MHz</span>
                    </div>
                </template>

                <!-- 下行频率列 -->
                <template #downlink_frequency="{row}">
                    <div v-if="row.product == 'BSC'">--</div>
                    <div v-else>
                        {{row.downlinkFrequency}}
                        <span v-if="row.downlinkFrequency">MHz</span>
                    </div>
                </template>

                <!-- Remark列头 -->
                <template #remark_header>
                    <div class="remarkHeaderCls">
                        <span v-if="!editingRemarkLabel" style="display: flex; align-items:center;">
                            <span :title="currentRemarkLabel" style="max-width: 140px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: middle;">
                                {{currentRemarkLabel}}
                            </span>
                            <i class="el-icon el-icon-operation-edit" @click="startEditRemarkLabel" style="margin-left: 5px;cursor: pointer;"></i>
                        </span>
                        <span v-else>
                            <el-input v-model="remarkLabelInput" size="mini" style="width: 150px;" maxlength="30" @keyup.enter.native="saveRemarkLabel">
                                <template slot="suffix">
                                    <i @click="saveRemarkLabel" class="el-icon el-icon-operation-defaultBeta" style="margin-right: 5px;cursor: pointer;"></i>
                                    <i @click="cancelEditRemarkLabel" class="el-icon el-icon-deactivate" style="cursor: pointer;"></i>
                                </template>
                            </el-input>
                        </span>
                    </div>
                </template>
            </vxe-grid>
        </div>

        <!-- 分页组件 -->
        <el-pagination
            small
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
            :current-page="pageParams.pageNum"
            :page-sizes="[ 50, 100, 200]"
            :page-size="pageParams.pageSize"
            layout="total, sizes, prev, pager, next, jumper,slot"
            :total="pageParams.total">
            <span @click="refreshList">
                <i class="el-icon el-icon-common-refresh" style="margin-left: 10px;cursor: pointer;line-height:22px;font-size:14px;"></i>
            </span>
        </el-pagination>
        
        <el-cmenu ref="menu" :data="menus" @click="menuClick"></el-cmenu>
        <div style="position: absolute;bottom: 8px;z-index: 100;display: flex;right: 100px;">
            <div class="status-statistic active" style="border-right: none;">
                <span class="el-icon el-icon-status-conn-on"></span>
                <span style="margin: 0 10px;"><%=rb.getString("ShiFouZaiXian")%></span> 
                <span id="gsm_online_count_rate" style="margin-left: 15px;"></span>
            </div>
            <!-- <div class="status-statistic active" style="border-right: none;">
                <span class="el-icon el-icon-status-active"></span>
                <span style="margin: 0 10px;"><%=rb.getString("JiHuo")%> </span> 
                <span id="gsm_active_count_rate" style="margin-left: 15px;"></span>
            </div> -->
        </div>

		<el-slide ref="settingPage" class="settingSlide" 
           	:url="settingUrl" width="80%"
            :footer="false" 
            :header="false">
        </el-slide>
		<el-slide ref="satelite" class="no-padding"
            title='<%=rb.getString("GPSWeiXingShu")%>'
            @cancel="closeSatellite"
            :footer="false" >
            <el-ctable :url="satelliteURL" :class="{loading: sateloading}">
                <el-table-column label='<%=rb.getString("WeiXingHao")%>' prop='gnssSvId'></el-table-column>
                <el-table-column label='<%=rb.getString("WeiXingXinHao")%>(dB-Hz)' prop='snr'></el-table-column>
            </el-ctable>
        </el-slide>
        <!-- sliders -->
        <el-slide ref="ueCount" :title="ueslide.title" :height='ueslide.height' :footer="false" @cancel="closeUeSlide" class='commonWarp'>
            <el-ctable :data="ueslide.data" :pagination="false">
                <el-table-column label="UEID" prop="ue_id" width="100"></el-table-column>
                <el-table-column label="IMSI" prop="imsi" width="150"></el-table-column>
                <el-table-column label="VMAC" prop="vmac" width="130"></el-table-column>
                <el-table-column label="<%=rb.getString("CPEName")%>" prop="cpe_name" width="150"></el-table-column>
                <el-table-column label="<%=rb.getString("XiaXingTunTuLv")%>" width="140" prop="downlink_rate"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingTunTuLv")%>" width="135" prop="uplink_rate"></el-table-column>
                <el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="ip" width="140"></el-table-column>
                <el-table-column label="<%=rb.getString("DuanKou")%>" prop="port" width="80"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingSinr")%>" prop="ulsinr" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingCqi")%>" prop="dlcqi" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_Dlcqi" prop="p_dlcqi" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_Dlcqi" prop="s_dlcqi" width="100"></el-table-column>

                <el-table-column label="<%=rb.getString("ShangXingmcs")%>" prop="ulmcs" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingmcs")%>" prop="dlmcs" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_Dlmcs" prop="p_dlmcs" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_Dlmcs" prop="s_dlmcs" width="100"></el-table-column>

                <el-table-column label="<%=rb.getString("FaSongGongLv")%>(dBm)' prop="txpower" width="100"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingbler")%>(%)' prop="uplink_bler" width="120"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_TB1_Downlink_BLER(%)" prop="p1_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="P_TB2_Downlink_BLER(%)" prop="p2_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_TB1_Downlink_BLER(%)" prop="s1_downlink_bler" width="165"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_TB2_Downlink_BLER(%)" prop="s2_downlink_bler" width="165"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingbler")%>(%)' prop="downlink_bler" width="165"></el-table-column>

                <el-table-column label="<%=rb.getString("LuJingSunHao")%>(dBm)' prop="pathloss" width="120"></el-table-column>
                <el-table-column v-if='ueS1apId == true' label="UE_S1AP_ID" prop="ue_s1ap_id" width="120"></el-table-column>
				<el-table-column v-if='mmeS1apId == true' label="MME_S1AP_ID" prop="mme_s1ap_id" width="120"></el-table-column>
            </el-ctable>
        </el-slide>

        <!--激活/取消激活 弹窗-->
		<el-dialog title="<%=rb.getString("QueRen")%>" id="gsmActiveCellDialog" :visible.sync="showActiveCellDialog" top="30vh" ref="activeCellDialog" 
			width="660" :close-on-click-modal="false"  @close='closeActiveCellDialog' append-to-body>
				<div style="margin-bottom:20px;">确认激活/取消激活？</div>
				<div style="border:1px solid #e9e9e9;padding:10px;min-height:100px;" v-loading='activeCellLialogLoading'>
						<div v-for="item in cellInfosList" class="borderCardItemCls" v-show='item.cellItem.length>0'>
							<span class="borderCardItemLabelCls">{{item.label}}（<%= rb.getString("BanKaID")%>）</span>
							<div v-for="(items,index) in item.cellItem"  style="display:inline-block;margin-right:10px;width:150px;font-size:12px;">
								<el-checkbox v-model="items.activeStatus" :true-label="1" :false-label="0">
									{{index+1}}
								</el-checkbox>
								<div v-if="items.oldStatus == '0'" style="display:inline-block;">
									(<span class="el-icon el-icon-status-active redIcon"></span><%= rb.getString("QuJiHuo")%>)
								</div>
								<div v-if="items.oldStatus == '1'" style="display:inline-block;">
									(<span class="el-icon el-icon-status-active greenIcon"></span><%= rb.getString("JiHuo")%>)
								</div>
							</div>
						</div>
				</div>
				<span slot="footer">
					<div>
						<el-button type="primary" @click="activeCellSubmit"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="closeActiveCellDialog"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</span>
		</el-dialog>

        <el-dialog title="<%=rb.getString("QueRen")%>" top="30vh" width="550"
            :visible.sync="collectMessageShow" 
            :modal="false"
            :close-on-click-modal="false">
            <el-form :model="collectForm">
                <div>{{confirmTips}}</div>
                <el-form-item label="<%=rb.getString("ChiXuShiChang")%>" style="display: flex;align-items: center;margin: 5px 0px;">
                    <el-select v-model="collectForm.collectInterval" placeholder="Select time" class="collect-select">
                        <el-option label="05" value="05"></el-option>
                        <el-option label="10" value="10"></el-option>
                    </el-select>
                    <div style="display: inline;padding: 5px;margin-left: -4px;border: 1px solid #e9e9e9;background: #F5F7FA;"><%=rb.getString("ANRFenZhong")%></div>
                </el-form-item>
                <span v-if="cllectExisted">
                    <span style="color: #B3B3B3;"><%=rb.getString("ShouJiBaoWenFuGaiTiShi")%> SN={{existedMsgSN}}. </span>
                </span>
            </el-form>

            <div slot="footer" style="text-align: right;">
                <el-button type="primary" @click="sendCollect"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="collectMessageShow = false"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </el-dialog>
        
        <el-dialog :title="'<%=rb.getString("YiDongDaoSheBeiZu")%>'" :visible.sync="deviceGroupMoveShow" width="620"
			:close-on-click-modal="false" top="30vh" :append-to-body="true">
            <el-ctable ref="ctableGroup" :url="deviceGroupUrl" id='gsmCtableGroup' @row-click="groupIdChange" :height="groupHeight" :row-key="'id'" pagination="true" :rownumber=true style='border: 1px solid #E9E9E9;'>
				<el-table-column label='' width="40">
					<template slot-scope="scope">
		           		<el-radio v-model="groupId" :label="scope.row.id"><span></span></el-radio>
		         	</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("SheBeiZuMingCheng")%>" min-width="150" prop="group_name"></el-table-column>
			</el-ctable>
			<div v-show="deviceGroupTip"><div slot="tip" class="el-upload__tip" ><%=rb.getString("QingXuanZeSheBeiZu")%> </div></div>
            <div slot="footer" style='text-align: right;'>
                <el-button type="primary" @click="deviceGroupMoveSubmit"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="deviceGroupMoveShow = false"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </el-dialog>

        <!-- 参数同步弹窗 -->
        <el-dialog title="<%=rb.getString("TongBu")%>" :visible.sync="openSyncDialogShow" :close-on-click-modal="false" @close="closeSyncDialog">
            <div id="gsm_sync-content" :class="{'loading': isSyncloading}" style="min-height: 200px;">
            </div>
        </el-dialog>
    </div>

    <script>
        var enbShowCols = '${showCol}',
            columncell = "${cellColumn}",//未选中的列表标识
            allColumn = "${allColumn}",
            halobSwitchFlag = '';

        var gsmvm = new Vue({
            el: '#gsmMonitorPage',
            data() {
                var vm = this;

                return {
                    // vxe-table 数据和状态
                    tableData: [],
                    tableLoading: false,
                    checkedRowKeys: [],
                    sortName: '',
                    sortOrder: '',
                    refreshTimer: null,
                    // 分页参数
                    pageParams: {
                        pageNum: 1,
                        pageSize: 50,
                        total: 0
                    },
                    
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
                        {field: 'serial_number', label: "<%=rb.getString("BSCBianMa")%>",sortable: true, width: 180},
                        {field: 'host_name', label: "<%=rb.getString("BSCMingCheng")%>",sortable: true, width: 150},
                        {field: 'op_state', label: "<%=rb.getString("ShiFouJiHuo")%>",sortable: true, width: 120},
                        {field: 'product', label: "<%=rb.getString("ChanPinLeiXingBiaoZhi")%>",sortable: true, width: 110},
                        {field: 'module_type', label: "<%=rb.getString("SheBeiXingHaoMing")%>",sortable: true, width: 120},
                        {field: 'product_name', label: "<%=rb.getString("ChanPinMingCheng")%>",sortable: true, width: 120},
                        {field: 'cell_ip', label: "<%=rb.getString("IPDiZhi")%>",sortable: true, width: 120},
                        {field: 'sub_station_name', label: '<%=rb.getString("ZhanZhiMingCheng")%>',width: 120},
                        {field: 'mac_address', label: 'MAC',sortable: true, width: 130},
                        {field: 'ue_count', label: "<%=rb.getString("UEShu")%>",sortable: true, width: 80},
                        {field: 'up_time', label: "<%=rb.getString("YunXingShiJian")%>",sortable: true, width: 120},
                        {field: 'online_duration', label: "<%=rb.getString("LeiJiShiChang")%>",sortable: true, width: 120},
                        {field: 'first_online_time', label: "<%=rb.getString("DiYiCiLianJieShiJian")%>",sortable: true, width: 140},
                        {field: 'LASTINFORMTIME', label: "<%=rb.getString("ShangCiLianJieShiJian")%>",sortable: true, width: 220},
                        {field: 'software_version', label: "<%=rb.getString("SoftwareVersion")%>",sortable: true, width: 140},
                        {field: 'firmware_version', label: "<%=rb.getString("FirmwareVersion")%>",sortable: true, width: 135},
                        {field: 'group_name', label: "<%=rb.getString("SheBeiZu")%>",sortable: true, width: 130},
                        {field: 'remark', label: 'Remark', width: 140},
                    ],
                    selectedRow: '',
                    menus: [],
                    settingUrl:'',
                    tbURL: '',
                    queryParams: {
                        sort: '',
                        order: '',
                        rows: 50,
                        page: 1,
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
                        title: "<%=rb.getString("XinXi")%>",
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
                        title: "<%=rb.getString("XiQuKeYongXiangQing")%>",
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
                    deviceGroupUrl:'${ctx}/pm/template/getTempDeviceGroupList.action',
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
                    enbAdditionalColShow:'${enbAdditionalColShow}',
                    batchSync:false,
                    batchCode:'',

                    operatorCtableDefaultCheckList: [],
                    operatorCtableCheckList:[],
                    operatorCtableUrl: '${ctx}/system/operator/getOperatorListByPage.action',
                    isExportDeviceLicenseInfo: false,
                    gsmExportFormatType: 'xlsx',
                    editingRemarkLabel: false,
                    remarkLabelInput: 'Remark',
                    currentRemarkLabel: 'Remark',
                    
                };
            },
            computed: {
                limitBatch(){
                    return batchOperation ? '' : 1;
                },
                confirmTips() {
                    var vm = this,
                        sn = vm.selectedRow.serial_number,
                        msg = "<%=rb.getString("QueRenShouJiPre")%>";

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
                isMonitorWritable() {
                    return writableMap['CODE_ENB_MONITOR'] == true;
                },
                // vxe-table 动态列配置
                tableColumns() {
                    var vm = this;
                    var cols = [];
                    
                    // 序号列 - 根据分页动态计算
                    cols.push({field: 'row_index', title: ' ', width: 60, fixed: 'left', align: 'center', slots: {default: 'row_index'}});
                    
                    // 复选框列
                    if (vm.enableCheckbox) {
                        cols.push({type: 'checkbox', width: 50, fixed: 'left'});
                    }

                    // 操作列
                    cols.push({field: 'enb_operation', title: '', width: 70, fixed: 'left', slots: {default: 'enb_operation'}});
                    
                    // 连接状态列
                    cols.push({field: 'connection_status', title: '', width: 40, fixed: 'left', sortable: true, slots: {default: 'connection_status'}});
                    
                    // 告警数量列
                    cols.push({field: 'alarm_count', title: "<%=rb.getString("GaoJingShu")%>", width: 110, fixed: 'left', sortable: true, slots: {default: 'alarm_count'}});
                    
                    // BSC编码列
                    cols.push({field: 'serial_number', title: "<%=rb.getString("BSCBianMa")%>", width: 180, fixed: 'left', sortable: true});
                    
                    // BSC名称列
                    cols.push({field: 'host_name', title: "<%=rb.getString("BSCMingCheng")%>", width: 150, fixed: 'left', sortable: true, slots: {default: 'host_name'}});
                    
                    // 射频开关状态列
                    if (vm.showColums.includes('rf_status')) {
                        cols.push({field: 'rf_status', title: "<%=rb.getString("ShePinKaiGuanZhuangTai")%>", width: 150, sortable: true, showOverflow: false, slots: {default: 'rf_status'}});
                    }
                    
                    // 是否激活列
                    if (vm.showColums.includes('op_state')) {
                        cols.push({field: 'op_state', title: "<%=rb.getString("ShiFouJiHuo")%>", width: 120, sortable: true, slots: {default: 'op_state'}});
                    }
                    
                    // 产品类型标识列
                    if (vm.showColums.includes('product')) {
                        cols.push({field: 'product', title: "<%=rb.getString("ChanPinLeiXingBiaoZhi")%>", width: 110, sortable: true, slots: {default: 'product'}});
                    }
                    
                    // 设备型号名列
                    if (vm.showColums.includes('module_type')) {
                        cols.push({field: 'module_type', title: "<%=rb.getString("SheBeiXingHaoMing")%>", width: 120, sortable: true, slots: {default: 'module_type'}});
                    }
                    
                    // 产品名称列
                    if (vm.showColums.includes('product_name')) {
                        cols.push({field: 'product_name', title: "<%=rb.getString("ChanPinMingCheng")%>", width: 120, sortable: true});
                    }
                    
                    // IP地址列
                    if (vm.showColums.includes('cell_ip')) {
                        cols.push({field: 'cell_ip', title: "<%=rb.getString("IPDiZhi")%>", width: 120, sortable: true, slots: {default: 'cell_ip'}});
                    }
                    
                    // 站址名称列
                    if (vm.showColums.includes('sub_station_name')) {
                        cols.push({field: 'sub_station_name', title: '<%=rb.getString("ZhanZhiMingCheng")%>', width: 120});
                    }
                    
                    // MAC列
                    if (vm.showColums.includes('mac_address')) {
                        cols.push({field: 'mac_address', title: 'MAC', width: 130, sortable: true});
                    }
                    
                    // UE数列
                    if (vm.showColums.includes('ue_count')) {
                        cols.push({field: 'ue_count', title: "<%=rb.getString("UEShu")%>", width: 80, sortable: true, slots: {default: 'ue_count'}});
                    }
                    
                    // Ipa Unit Id列
                    if (vm.showColums.includes('IpaUnitId')) {
                        cols.push({field: 'IpaUnitId', title: 'Ipa Unit Id', width: 120, slots: {default: 'ipa_unit_id'}});
                    }
                    
                    // Oml Remote Ip列
                    if (vm.showColums.includes('OmlRemoteIp')) {
                        cols.push({field: 'OmlRemoteIp', title: 'Oml Remote Ip', width: 120, slots: {default: 'oml_remote_ip'}});
                    }
                    
                    // Oml Remote Ip Bak列
                    if (vm.showColums.includes('OmlRemoteIpBak')) {
                        cols.push({field: 'OmlRemoteIpBak', title: 'Oml Remote Ip Bak', width: 150, slots: {default: 'oml_remote_ip_bak'}});
                    }
                    
                    // BSC Select列
                    if (vm.showColums.includes('BscSelect')) {
                        cols.push({field: 'BscSelect', title: 'BSC Select', width: 100, slots: {default: 'bsc_select'}});
                    }
                    
                    // BSC连接状态列
                    if (vm.showColums.includes('BscLinkStatus')) {
                        cols.push({field: 'BscLinkStatus', title: "<%=rb.getString("BSCLianJieZhuangTai")%>", width: 120, slots: {default: 'bsc_link_status'}});
                    }
                    
                    // 所属BSC编码列
                    if (vm.showColums.includes('BSCSerialNumber')) {
                        cols.push({field: 'BSCSerialNumber', title: "<%=rb.getString("SuoShuBSCBianMa")%>", width: 160, slots: {default: 'bsc_serial_number'}});
                    }
                    
                    // BTS数列
                    if (vm.showColums.includes('BtsNum')) {
                        cols.push({field: 'BtsNum', title: "<%=rb.getString("BTSShu")%>", width: 90, slots: {default: 'bts_num'}});
                    }
                    
                    // 同步状态列
                    if (vm.showColums.includes('synStatus')) {
                        cols.push({field: 'synStatus', title: "<%=rb.getString("TongBuZhuangTai")%>", width: 160, sortable: true, slots: {default: 'syn_status'}});
                    }
                    
                    // GPS卫星数列
                    if (vm.showColums.includes('gps_satellite_count')) {
                        cols.push({field: 'gps_satellite_count', title: "<%=rb.getString("GPSWeiXingShu")%>", width: 85, sortable: true, slots: {default: 'gps_satellite_count'}});
                    }
                    
                    // GPS经度列
                    if (vm.showColums.includes('gps_longitude')) {
                        cols.push({field: 'gps_longitude', title: "<%=rb.getString("GPSJingDu")%>", width: 100, slots: {default: 'gps_longitude'}});
                    }
                    
                    // GPS纬度列
                    if (vm.showColums.includes('gps_latitude')) {
                        cols.push({field: 'gps_latitude', title: "<%=rb.getString("GPSWeiDu")%>", width: 90, slots: {default: 'gps_latitude'}});
                    }
                    
                    // GPS高度列
                    if (vm.showColums.includes('gps_height')) {
                        cols.push({field: 'gps_height', title: "<%=rb.getString("GPSGaoDu")%>", width: 70, slots: {default: 'gps_height'}});
                    }
                    
                    // LAC列
                    if (vm.showColums.includes('currentLac')) {
                        cols.push({field: 'currentLac', title: 'LAC', width: 70, slots: {default: 'current_lac'}});
                    }
                    
                    // 频点列
                    if (vm.showColums.includes('currentArfcn')) {
                        cols.push({field: 'currentArfcn', title: "<%=rb.getString("PinDian")%>", width: 70, slots: {default: 'current_arfcn'}});
                    }
                    
                    // 上行频率列
                    if (vm.showColums.includes('uplinkFrequency')) {
                        cols.push({field: 'uplinkFrequency', title: "<%=rb.getString("ShangXingPinLv")%>", width: 110, slots: {default: 'uplink_frequency'}});
                    }
                    
                    // 下行频率列
                    if (vm.showColums.includes('downlinkFrequency')) {
                        cols.push({field: 'downlinkFrequency', title: "<%=rb.getString("XiaXingPinLv")%>", width: 110, slots: {default: 'downlink_frequency'}});
                    }
                    
                    // 运行时间列
                    if (vm.showColums.includes('up_time')) {
                        cols.push({field: 'up_time', title: "<%=rb.getString("YunXingShiJian")%>", width: 120, sortable: true});
                    }
                    
                    // 累计时长列
                    if (vm.showColums.includes('online_duration')) {
                        cols.push({field: 'online_duration', title: "<%=rb.getString("LeiJiShiChang")%>", width: 120, sortable: true});
                    }
                    
                    // 第一次连接时间列
                    if (vm.showColums.includes('first_online_time')) {
                        cols.push({field: 'first_online_time', title: "<%=rb.getString("DiYiCiLianJieShiJian")%>", width: 140, sortable: true});
                    }
                    
                    // 上次连接时间列
                    if (vm.showColums.includes('LASTINFORMTIME')) {
                        cols.push({field: 'LASTINFORMTIME', title: "<%=rb.getString("ShangCiLianJieShiJian")%>", width: 220, sortable: true});
                    }
                    
                    // 软件版本列
                    if (vm.showColums.includes('software_version')) {
                        cols.push({field: 'software_version', title: "<%=rb.getString("SoftwareVersion")%>", width: 140, sortable: true});
                    }
                    
                    // 固件版本列
                    if (vm.showColums.includes('firmware_version')) {
                        cols.push({field: 'firmware_version', title: "<%=rb.getString("FirmwareVersion")%>", width: 135, sortable: true});
                    }
                    
                    // 设备组列
                    if (vm.showColums.includes('group_name')) {
                        cols.push({field: 'group_name', title: "<%=rb.getString("SheBeiZu")%>", width: 130, sortable: true});
                    }
                    
                    // Remark列
                    if (vm.showColums.includes('remark')) {
                        cols.push({field: 'remark', title: '', width: 185, slots: {header: 'remark_header'}});
                    }
                    
                    return cols;
                }
            },
            watch:{
                selectedRows(newVal){
                    if(newVal.length == 0){
                        this.bulkSelectShow = false;
                    }
                },
                queryParams: {
                    handler: function(newVal, oldVal) {
                        var vm = this;
                        // 查询参数变化时，重置到第一页
                        vm.pageParams.currentPage = 1;
                        vm.queryParams.page = 1;
                        vm.loadTableData();
                    },
                    deep: true
                }
            },
            methods: {
                initTb() {
                    var vm = this;

                    vm.tbURL = '${ctx}/cell/cpeinfos/queryCpeInfosList.action?monitor=1';
                    vm.loadTableData();
                    // 启动定时刷新
                    vm.startAutoRefresh();
                },
                // 加载表格数据
                loadTableData() {
                    var vm = this;
                    vm.tableLoading = true;
                    
                    var params = Object.assign({}, vm.queryParams, {
                        // 分页参数
                        page: vm.pageParams.pageNum,
                        rows: vm.pageParams.pageSize
                    });
                    
                    axios.post(vm.tbURL, stringify(params)).then(function(response) {
                        var data = response.data;
                        if (data && data.rows) {
                            vm.tableData = data.rows;
                            // 更新总数
                            vm.pageParams.total = data.total || data.rows.length;
                        } else if (Array.isArray(data)) {
                            vm.tableData = data;
                            vm.pageParams.total = data.length;
                        } else {
                            vm.tableData = [];
                            vm.pageParams.total = 0;
                        }
                        vm.tableLoading = false;
                        vm.loadSuccess(vm.tableData);
                    }).catch(function(error) {
                        vm.tableLoading = false;
                        console.error('[GSM Monitor] 加载数据失败:', error);
                    });
                },
                // 启动自动刷新
                startAutoRefresh() {
                    var vm = this;
                    if (vm.refreshTimer) {
                        clearInterval(vm.refreshTimer);
                    }
                    vm.refreshTimer = setInterval(function() {
                        vm.loadTableData();
                    }, 6000); // 6秒刷新一次
                },
                // 停止自动刷新
                stopAutoRefresh() {
                    var vm = this;
                    if (vm.refreshTimer) {
                        clearInterval(vm.refreshTimer);
                        vm.refreshTimer = null;
                    }
                },
                // 排序变化处理
                handleSortChange({ field, order }) {
                    var vm = this;
                    // 将排序参数映射到 queryParams
                    vm.queryParams.sort = field || '';
                    // vxe-table order: 'asc'/'desc'/null, 后端需要: 'asc'/'desc'/''
                    vm.queryParams.order = order || '';
                    // 排序时重置到第一页
                    vm.pageParams.currentPage = 1;
                    vm.queryParams.page = 1;
                },
                // 选择变化处理
                handleSelectionChange(params) {
                    var vm = this;
                    var records = vm.$refs.list.getCheckboxRecords();
                    vm.selectedRows = records || [];
                },
                // 判断行是否可选
                checkSelectable(params) {
                    var row = params.row;
                    if (row.dual_carrier_type == 2 && row.product != 'RTD') {
                        return false;
                    }
                    return true;
                },
                // 获取表格数据
                getData() {
                    return this.tableData;
                },
                // 分页大小变化处理
                handleSizeChange(val) {
                    var vm = this;
                    vm.pageParams.pageSize = val;
                    vm.pageParams.pageNum = 1; // 重置到第一页
                    vm.loadTableData();
                },
                // 当前页变化处理
                handleCurrentChange(val) {
                    var vm = this;
                    vm.pageParams.pageNum = val;
                    vm.loadTableData();
                },
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

                    axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
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

                    axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
                        var data = res.data;

                        if(data && data.isExist == 'true') {
							vm.$message.error('SN=' + data.serialNumber + "<%=rb.getString("ZhengZaiShouJi")%>");
                        }else {
                            axios.post('${ctx}/trace/start.action', stringify(params)).then(function(res){
                                var data = res.data;

                                if(data.success == true) {
                                    queryGsmVue.startInterval(time);
                                    queryGsmVue.queryLatestInfo();
                                    vm.collectMessageShow = false;
                                    
                                    vm.$message.success("<%=rb.getString("ChengGong")%>");
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
                    this.loadTableData();
                },
                loadSuccess(data) {
                	this.refresh_cellStatusStatistics();
                },
                showAddOrImport() {
                    
                    $('.addGsmOrImport-content').html('');
                    $('.addGsmOrImport-content').each(function(idx,item){
                        if($(item).is(':visible')) {
                            $(item).load('${ctx}/cell/cpeinfos/toGSMMonitorAddPages.action',function(html) {
                               
                            })
                        }
                    })
                },
                // 获取所有列
                getAllDefaultCols() {
                    var vm = this,
                        columns = [
                            {label: "<%=rb.getString("BSCBianMa")%>", prop: 'serial_number', width: 180},
                            {label: "<%=rb.getString("BSCMingCheng")%>", prop: 'host_name', width: 150, formatter: vm.nameFmt},
                            {label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>', prop: 'rf_status', width: 95},
                            {label: '<%=rb.getString("ShiFouJiHuo")%>', prop: 'op_state', width: 120},
                            {label: "<%=rb.getString("UEShu")%>", prop: 'ue_count', width: 80},
                            {label: "<%=rb.getString("IPDiZhi")%>", prop: 'cell_ip', width: 100},
                            {label: 'MAC', prop: 'mac_address', width: 115},
                        ];

                    if(supportTopoSite) {
                        columns.push({label: '<%=rb.getString("ZhanZhiMingCheng")%>', prop: 'sub_station_name', width: 110});
                    }

                    columns.push({label: "<%=rb.getString("ChanPinLeiXingBiaoZhi")%>", prop: 'product', width: 110});
                    columns.push({label: "<%=rb.getString("ChanPinMingCheng")%>", prop: 'product_name', width: 110});

                    columns.push({label: "<%=rb.getString("SheBeiXingHaoMing")%>", prop: 'module_type', width: 95});
                    columns.push({label: "<%=rb.getString("SoftwareVersion")%>", prop: 'software_version', width: 120});
                    columns.push({label: "<%=rb.getString("SheBeiZu")%>", prop: 'group_name', width: 130});
                    
                    columns.push({label: "<%=rb.getString("YunXingShiJian")%>", prop: 'up_time', width: 110});
                    
                    columns.push({label: "<%=rb.getString("LeiJiShiChang")%>", prop: 'online_duration', width: 110});
                    columns.push({label: "<%=rb.getString("DiYiCiLianJieShiJian")%>", prop: 'first_online_time', width: 125});
                    columns.push({label: "<%=rb.getString("ShangCiLianJieShiJian")%>", prop: 'LASTINFORMTIME', width: 220});
                    columns.push({label: "<%=rb.getString("FirmwareVersion")%>", prop: 'firmware_version', width: 115});

                    columns.push({label: "Ipa Unit Id", prop: 'IpaUnitId', width: 120});
                    columns.push({label: "Oml Remote Ip", prop: 'OmlRemoteIp', width: 120});
                    columns.push({label: "Oml RemoteIp Bak", prop: 'OmlRemoteIpBak', width: 150});
                    columns.push({label: "BSC Select", prop: 'BscSelect', width: 100});
                    columns.push({label: "BSC Link Status", prop: 'BscLinkStatus', width: 120});
                    columns.push({label: "BSC SerialNumber", prop: 'BSCSerialNumber', width: 150});
                    columns.push({label: "BTS Num", prop: 'BtsNum', width: 90});
                    columns.push({label: '<%=rb.getString("TongBuZhuangTai")%>', prop: 'synStatus', width: 160});
                    columns.push({label: '<%=rb.getString("GPSJingDu")%>', prop: 'gps_longitude', width: 100});
                    columns.push({label: '<%=rb.getString("GPSWeiDu")%>', prop: 'gps_latitude', width: 90});
                    columns.push({label: '<%=rb.getString("GPSGaoDu")%>', prop: 'gps_height', width: 70});
                    columns.push({label: '<%=rb.getString("GPSWeiXingShu")%>', prop: 'gps_satellite_count', width: 75});

                    columns.push({label: 'LAC', prop: 'currentLac', width: 70});
                    columns.push({label: '<%=rb.getString("PinDian")%>', prop: 'currentArfcn', width: 70});
                    columns.push({label: '<%=rb.getString("ShangXingPinLv")%>', prop: 'uplinkFrequency', width: 110});
                    columns.push({label: '<%=rb.getString("XiaXingPinLv")%>', prop: 'downlinkFrequency', width: 110});
                    columns.push({label: 'Remark', prop: 'remark', width: 125});
                    // columns.push({label: "<%=rb.getString("JianQuanMa")%>", prop: 'authCode', width: 120});

                    return columns;
                },
                selectionChange(s) {
                    this.selectedRows = s||[];
                },
                closeSyncName() {
                    document.body.click();
                },
                clearSelection() {
                	var vm = this;
                	if (vm.$refs.list) {
                	    vm.$refs.list.clearCheckboxRow();
                	    vm.$refs.list.clearCheckboxReserve();
                	}
                	vm.selectedRows = [];
                },
                toggleRowSelection(row, selected) {
                    var vm = this;
                    if (vm.$refs.list) {
                        if (selected) {
                            vm.$refs.list.setCheckboxRow(row, true);
                        } else {
                            vm.$refs.list.setCheckboxRow(row, false);
                        }
                    }
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
    	    		axios.post('${ctx}/system/deviceGroup/moveCellToGroup.action',stringify(params)).then(function(response){
    					let data = response.data;
    					if(data.success){
    						vm.$message({
    							message: "<%=rb.getString("ChengGong")%>",
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
                		rows = vm.getData(),
            			cellCodes = '';

                	if(vm.selectedRows.length == 0)return

                	/*rows.map(function(row){
                		if(codes.includes(row.serial_number)) {
                			if(row.connection_status != 'Off'){
                				row.connection_status = 'updating';
                			}
                		}
                	});*/
                	
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
            		
            		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiSheBei")%>", function (r) {
            			if (r) {
            				showMsg('prompt_msg',"<%=rb.getString("MingLingYiXiaFa")%>")
            				var param = {
            					cellCodes: cellCodes
            				};
            				$.post("${ctx}/task/reboot/batchRebootCell.action", param, function (data) {
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

                    $.post('${ctx}/cell/cpeinfos/syncCellName.action?smallCellCode='+code, function(data){
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
                        url: '${ctx}/cell/topo/syncGPSInfo.action',
                        type: 'post',
                        data: {cell_code: code},
                        dataType: 'json',
                        success: function(data) {
                            if(data.success){
                                vm.refreshList();
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
                    if ("${carrierType}" == "1") {
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

                    // 检查 eNBTopo_tab.jsp 是否已打开 setting.jsp 页面，如果打开则先关闭以防止 ID 冲突
                    if(typeof topovm !== 'undefined') {
                        try {
                            // 直接尝试关闭 eNBTopo_tab 的 slide，即使它没有打开也不会报错
                            topovm.$refs.topoSettingSlide.hide();
                        } catch(e) {}
                    }
                	this.settingUrl = '${ctx}/cell/cpeinfos/toGSMMonitorSettingPages.action';
                	this.$refs.settingPage.showSlide(function(){
                		eventBus.$emit("gsm-setting-row",row,page,'monitor')
                    });
                },
				closeSettingPage(){
                	this.$refs.settingPage.hide();
                    this.$refs.list.refresh();
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
                    var XinXi = "<%=rb.getString("XinXi")%>";
                    var TongBu = "<%=rb.getString("TongBu")%>";
                    var SheZhi = "<%=rb.getString("SheZhi")%>";
                    var MeiYouQuanXian = "<%=rb.getString("MeiYouQuanXian")%>";
                    var ChongQi = "<%=rb.getString("ChongQi")%>";
                    var RiZhi = "<%=rb.getString("RiZhiShouJi")%>";
                    var MiMaChongZhi = "<%=rb.getString("XiuGaiMiMa")%>";
                    var License = "<%=rb.getString("License")%>";
                    var CaoZuo = "<%=rb.getString("CaoZuo")%>";
                    var registerSAS = "<%=rb.getString("SasZhuCe")%>"; 
                    var deregisterSAS = "<%=rb.getString("ZhuXiaoSAS")%>";
                    var GengDuoCaoZuo = "<%=rb.getString("GengDuoCaoZuo")%>";
                    var WeiHuCaoZuo = "<%=rb.getString("Maintenance")%>";
                    var PeiZhiHuiFu = "<%=rb.getString("PeiZhiHuiFu")%>";
                    var RFname = "<%=rb.getString("ShePinCaoZuo")%>";
                    var RFon = "<%=rb.getString("Kai")%>";
                    var RFoff = "<%=rb.getString("Guan")%>";
                    var YouXiaoQi = "<%=rb.getString("YouXiaoQi")%>";
                    var LiuLiangXianZhi = "<%=rb.getString("LiuLiangXianZhi")%>";
                    var FenBuShi = "<%=rb.getString("FenBuShi")%>";
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
                        url:'${ctx}/cell/cpeinfos/getENBOperationItem.action',
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
                        {row: row, label:'<%=rb.getString("ShouJiBaoWen")%>',id:50,show: collectShow},
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
                            axios.post('${ctx}/pm/nxp/getSlotCellInfos.action',stringify({smallCellCode:smallcellCode})).then(function(response){
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
                        url = '${ctx}/cell/cpeinfos/cellModifyActiveStatus.action',
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
                            vm.$message.success("<%=rb.getString("ChengGong")%>")
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
                    axios.post('${ctx}/pm/nxp/getSlotCellInfos.action',stringify({smallCellCode:row.small_cell_code})).then(function(response){
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

			        vm.clearSelection();
                },
                // 设备已选表格 单个删除事件
                delBulkSelected(rows){
                    var vm = this,
                        rowKey = 'serial_number';
                    vm.selectedRows = vm.selectedRows.filter((items)=>{
                        return items[rowKey] != rows[rowKey]
                    });
                    // 从vxe-table中取消选中该行
                    vm.toggleRowSelection(rows, false);
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
                    loadHTML(document.querySelector('#gsmtoolbar_tableHomeCellList'),{
                        url: '${ctx}/cell/cpeinfos/toGSMMonitorQueryPages.action',
                        success: function() {}
                    });
                    vm.getCustomLabelData();
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
                                $(item).load('${ctx}/cell/cpeinfos/toGSMMonitorSyncParamsPages.action',function(html) {
                                    vm.isSyncloading = false;
                                })
                            }
                        })
                    },200);
                },
                showGsmExport(){
                    var vm = this;
                    
                    vm.$nextTick(function(){
                        vm.gsmExportFormatType = 'xlsx';
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
                    if(type == 'monitor_xlsx' || type == 'monitor_csv'){
                        var url = '${ctx}/cell/cpeinfos/exportGSMInfosToExcel.action';
                        if(type == 'monitor_csv') {
                            url = '${ctx}/cell/cpeinfos/exportGSMInfosToCsv.action';
                        }
                        downLoadFileByAxios(url, params);
                        
                        if(type == 'monitor_xlsx') queryGsmVue.queryProgress();

                    }else if(type == 'license'){
                        params.isGSM = 1;
                        downLoadFileByAxios('${ctx}/cell/cpeinfos/exportEnodebLicenseInfos.action', params);
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

                    var url = '${ctx}/cell/cpeinfos/exportGSMInfosToExcel.action';
                    if(vm.gsmExportFormatType == 'csv') {
                        url = '${ctx}/cell/cpeinfos/exportGSMInfosToCsv.action';
                    }
                    exportByForm(url, params);
                    setTimeout(function(){
                        if(vm.isExportDeviceLicenseInfo){
                            params.isGSM = 1;
                            exportByForm('${ctx}/cell/cpeinfos/exportEnodebLicenseInfos.action', params);
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
                    
                    $.post("${ctx}/cell/cpeinfos/getCellStatusStatistics.action", params, function(data) {
                        if(!data["connection_status"]){
                            connectionStatus = "0/0";
                            connectionStatusRef = "0/0"
                        }else{
                            connectionStatus = data["connection_status"];
                            connectionStatusRef = data["connection_status_ref"];
                        }
                        $('#gsm_online_count_rate').text("( "+connectionStatus+" )");
                         /*if(!data["op_state"]){
                            opStateStatus = "0/0";
                            opStateStatusRef = "0/0";
                        }else{
                            opStateStatus = data["op_state"];
                            opStateStatusRef = data["opState_ref"];
                        }
                        $('#gsm_active_count_rate').text("( "+opStateStatus+" )"); */
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
                    
                    $.post("${ctx}/cell/collect/goImmediateCollectLogFile.action", param, function (data) {
                        if (data["success"]) {
                            showMsg('success_msg','<%=rb.getString("RiZhiZhengZaiShouJi")%>')
                        } else {
                            showMsg('error_msg',data['message']);
                        }
                    }, "json");
                },
                getGsmGPSSignalData(rowData) {
                    var vm = this,
                        codes = rowData.small_cell_code,
                        url = "${ctx}/cell/param/getBTSSatellitesDataList.action?timeZone="+timeZone+'&smallCellCode='+codes+'&rd='+Math.random();
                    
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
                    } else if (value == ("GPS " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="GPS "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<span>"+(val)+"</span>"
                    }else if (value == ("1588 " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="1588 "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<span>"+(val)+"</span>"
                    }else if (value == ("REM " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="REM "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<span>"+(val)+"</span>"
                    }else if (value == "GPS "+ "<%= rb.getString("TongBuChengGong")%>" ) {
                        val ="GPS "+ "<%= rb.getString("TongBuChengGong")%>";
                        value = "<span>"+(val)+"</span>"
                    }else if (value == "1588 "+"<%= rb.getString("TongBuChengGong")%>" ) {	
                        val = "<%= rb.getString("TongBuChengGong")%>";
                        val = "1588 " + val;
                        value = "<span>"+(val)+"</span>"
                    }else if (value == "REM "+"<%= rb.getString("TongBuChengGong")%>") {	
                        val = "<%= rb.getString("TongBuChengGong")%>";
                        val = "REM " + val;
                        value = "<span>"+(val)+"</span>"
                    }else if (value == "<%= rb.getString("WeiTongBu")%>") {	
                        val = "<%= rb.getString("WeiTongBu")%>";
                        value = "<span class='offStatusCls'>"+(val)+"</span>"
                    }
                    return '<div style="min-width: 130px;diplay: line-block;">'+value+'</div>';
                },
                // 获取自定义label信息
                getCustomLabelData() {
                    var vm = this;

                    axios.post('${ctx}/cell/columnAlias/queryColumnAliasConfigs.action').then(function(response){
                        var data = response.data || [];

                        data.forEach(function(item){
                            if(item.columnName == 'remark'){
                                vm.currentRemarkLabel = item.columnAlias || 'Remark';
                                vm.remarkLabelInput = vm.currentRemarkLabel;
                            }
                        });
                    }).catch(function(error){});
                },
                // Remark label 编辑方法
                startEditRemarkLabel() {
                    this.editingRemarkLabel = true;
                    this.remarkLabelInput = this.currentRemarkLabel;
                },
                saveRemarkLabel() {
                    var vm = this,
                        params = {
                            columnName: 'remark',
                            columnAlias: this.remarkLabelInput.trim()
                        };
                    if(this.remarkLabelInput.trim() === ''){
                        vm.$message.warning('<%=rb.getString("QingShuRuBiTianXiang")%>');
                        return;
                    }
                    let paramsData = JSON.stringify(params);
                    axios.post('${ctx}/cell/columnAlias/setColumnAlias.action',paramsData,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
                        var data = response.data;

                        if(data.success){
                            vm.currentRemarkLabel = vm.remarkLabelInput.trim();
                            vm.editingRemarkLabel = false;
                            vm.syncRemarkLabel();
                            vm.$message.success('<%=rb.getString("ChengGong")%>');
                        }else{
                            vm.$message.error(data.message);
                        }
                    }).catch(function(error){});
                },
                cancelEditRemarkLabel() {
                    this.editingRemarkLabel = false;
                    this.remarkLabelInput = this.currentRemarkLabel;
                },
                // 同步其他页面的Remark label
                syncRemarkLabel() {
                    var vm = this,
                        newLabel = vm.currentRemarkLabel,
                        vueInstances = [
                            {name: 'enbvm', instance: typeof enbvm !== 'undefined' ? enbvm : null},
                            {name: 'gnbMonitor', instance: typeof gnbMonitor !== 'undefined' ? gnbMonitor : null},
                            {name: 'egwRegisterVue', instance: typeof egwRegisterVue !== 'undefined' ? egwRegisterVue : null},
                            {name: 'gnbOverviewVue', instance: typeof gnbOverviewVue !== 'undefined' ? gnbOverviewVue : null},
                            {name: 'enbDetailVue', instance: typeof enbDetailVue !== 'undefined' ? enbDetailVue : null},
                            {name: 'gsmSettingOverviewVue', instance: typeof gsmSettingOverviewVue !== 'undefined' ? gsmSettingOverviewVue : null},
                        ];

                    vueInstances.forEach(function(vue){
                        if(vue.instance && vue.instance.currentRemarkLabel !== undefined){
                            vue.instance.currentRemarkLabel = newLabel;
                        }
                        if(vue.instance && vue.instance.remarkLabelInput !== undefined){
                            vue.instance.remarkLabelInput = newLabel;
                        }
                    });
                },
            },
            created() {},
            mounted() {
                // 排序
                var vm = this,
                    sortCodes = enbShowCols.split(','),
                    sortList = vm.columns;
                vm.init();
                
                eventBus.$on('cancel-gsm-setting',vm.closeSettingPage);
            },
            beforeDestroy() {
                // 清理定时器
                this.stopAutoRefresh();
            }
        });
    </script>
</body>
</html>