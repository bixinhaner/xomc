<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>

<style type="text/css">
	#configurationPlan .importCard .w400{
		width:320px;
	}
	#configurationPlan .importCard .el-form-item{
		margin-bottom:16px;
	}
	#configurationPlan .issueCard .el-dialog__body,
	#configurationPlan .importCard .el-dialog__body{
		padding:30px !important;
		background:#FFFFFF;
		border:none;
	}
	#configurationPlan .issueCard .el-form-item__label,
	#configurationPlan .importCard .el-form-item__label{
		line-height:26px;
	}
	#configurationPlan .importCard .el-input__suffix{
		top:4px;
	}
	#configurationPlan .issueCard .fileAcceptTip,
	#configurationPlan .importCard .fileAcceptTip{
		color:#999999;
		font-size:12px;
		margin-left:16px;
	}
	#configurationPlan .issueCard .el-icon-circle-info:before,
	#configurationPlan .importCard .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	#configurationPlan .issueCard .el-form-item{
		margin-bottom:-6px;
	}
	#configurationPlan .el-checkbox__input.is-checked+.el-checkbox__label{
		color:#333;
		font-weight:normal;
	}
	#configurationPlan .successStatus:before{
		color:#67D972;
	}
	#configurationPlan .failStatus:before{
		color:#E88282;
	}
	#configurationPlan .wsxStatus:before{
		color:#4D84FF;
	}
	#configurationPlan .el-radio.is-bordered { min-width: 50px; height: 30px; padding: 7px 10px; }
	#configurationPlan .el-radio-group .el-radio__label { font-size: 12px; }
	#configurationPlan {
		width:100%;
		height:100%;
		display:flex;
		flex-direction:column;
	}
	#configurationPlan .enbDeviceWarp,
	#configurationPlan .taskListWarp{
		flex:1;
		overflow:auto;
	}
	#configurationPlan .taskListWarp {
		position:absolute;
		bottom:0;
		width:100%;
		height:350px;
		z-index:99;
	}
	#configurationPlan .taskListAllWarp {
		flex:1;
		overflow:auto;
		position:absolute;
		top:0;
		width:100%;
		height:100%;
		z-index:999;
	}
	#configurationPlan .queryGroup {
		height: 24px !important;
	}
	#configurationPlan .queryGroup .el-input,
	#configurationPlan .queryGroup input {
		height: 24px !important;
		width: 200px;
	}
	#configurationPlan .el-icon-common-search {
		font-size: 14px;
	}
	#configurationPlan .rightWarpLayerContent .el-form .el-form-item {
		margin-bottom: 20px;
	}
	#configurationPlan .switchItem .el-form-item__label {
		width: 80px;
		line-height: 20px;
	}
	#configurationPlan .retryInput .el-input__inner {
		width: 242px;
	}
	#configurationPlan .rightWarp .el-form-item__error {
		padding-top: 2px;
	}
	#configurationPlan .rightWarp .el-select .el-input {
		width: 230px;
	}
	#configurationPlan .showHighlightColor {
		color: #FF4614;
	}
	#configurationPlan .hideHighlightColor {
		color: rgba(0, 0, 0, 0.6);		
	}
	#configurationPlan .rightImportBox .el-input{ width: 320px } 
	#configurationPlan .rightWarp .el-input-group__append { margin: 0 8px; background-color: #FFFFFF; }
	#configurationPlan .rightWarp .el-input-group__append .el-icon { font-size: 14px; }
	#configurationPlan .rightWarp .el-input-group__append .el-icon:before { color: #7A7992; }
	#configurationPlan .commonRadioButton .batchRadio,
	#configurationPlan .commonRadioButton .batchRadio .el-radio-button__inner {
		width: 126px;
	}
	#configurationPlan .disabledCheck {
		color: #7A7992 !important;
		cursor: not-allowed !important;
	}
	#configurationPlan .disabledCheck .el-icon:before{
		opacity: 0.3;
		cursor: not-allowed !important;
	}
	#configurationPlan .closeIconBox .el-icon-close {
		color: #7A7992;
	}
	.enbExportContent>div {
		margin-bottom:10px;
	}
	.enbExportContent .el-checkbox-group {
		display:flex;
		flex-wrap:wrap;
	}
	.enbExportContent .el-checkbox-group>label{
		width:22%;
		margin-top:8px;
	}
	#configurationPlan .commonTable .el-ctable-toolbar {
		padding: 22px 0 !important; 
	}
    #configurationPlan .flex-item-cls {
        height: 100%;
        overflow: auto;   
        position: relative;
        box-sizing: border-box;
    }
	.exportTipCls[tip]:hover::after  {
		right: 0px;
	}
</style>
<!--配置 页面 -->
<div class="panelDefault" id="configurationPlan" style='border:none; background: #F6F7FB;'>
    <el-tabs v-model="tabActiveName" class='newTabs' style="height: 100%;">
        <el-tab-pane label='<%=rb.getString("JiZhan")%>' name="eNB">
            <div class='commonFlex' style='width: 100%; height: 100%;'>
                <div class='leftWarp' style='background: #F6F7FB;'>
                    <div class="enbDeviceWarp" style='background: #FFFFFF;'>
                        <div class='toolbarHeadBtnBoxCls'>
                            <div class="selectBlukBoxCls" v-show='hasConfigRole'>
                                <div class="selectMain">
                                    <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                        <span class="el-icon-operation-defaultBeta el-icon"></span>
                                        <span class="bulkSelectNumBoxCls">( {{selectionData.length}} )</span>
                                    </div>
                                    <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 100px;" v-show="bulkSelectShow">
                                        <div class="selectBoxTitle">
                                            <span><%=rb.getString("YiXuan")%></span>
                                            <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable()"></span>
                                        </div>
                                        <div class="selectBoxMain">
                                            <div class="tableInfoCls">
                                                <div class="tableInfoHeader">
                                                        <div><%=rb.getString("Title_SheBeiBianMa")%></div>
                                                    <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
                                                </div>
                                                <el-ctable 
                                                    id="bulkSelectTable" 
                                                    ref="bulkSelectTable" 
                                                    :data="selectionData" 
                                                    :showHeader="false"
                                                    :rownumber="false"
                                                    :front-pagination="true"
                                                    height="270px" pagination="true" >
                                                    <el-table-column prop="id" v-if="false"></el-table-column>
                                                    <el-table-column width="588">
                                                        <template slot-scope="scope" >
                                                            <div class="tableItemCls">
                                                                <span v-if='activeName=="neighborFrequencyConfig" || activeName=="neighborCellConfig"'>{{scope.row.serialNumber}}</span>
                                                                <span v-else>{{scope.row.serial_number}}</span>
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
                            <!-- 下发参数 -->
                            <div class='commonFlex'>
                                <div v-if="hasConfigRole" :class="selectionData.length > 0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="issueBtn">
                                    <span class="el-icon el-icon-operation-issued"></span>
                                    <span><%=rb.getString("PeiZhiXiaFa")%></span>
                                </div>
                                <div v-if="hasConfigRole" :class="selectionData.length > 0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="deleteCells" style='border: none;'>
                                    <span class="el-icon el-icon-operation-delete"></span>
                                    <span><%=rb.getString("PiLiangShanChu")%></span>
                                </div>
                            </div>
                            <div v-show='activeName == "batch" && hasConfigRole' @click="issueBtn('all')" class="newIconBoxCls-bt" style="right:225px;top:5px;" tip="<%=rb.getString("KaiShiSuoYou")%>">
                                <span class="el-icon-operation-start el-icon"></span>
                            </div>
                            <div v-show='activeName == "batch" && hasConfigRole' @click="terminateIssueBtn" class="newIconBoxCls-bt" style="right:190px;top:5px;" tip="<%=rb.getString("TingZhiSuoYou")%>">
                                <span class="el-icon-operation-terminate el-icon"></span>
                            </div>
                            <div v-show='activeName == "batch" && hasConfigRole' @click="deleteCells('all')" class="newIconBoxCls-bt" style="right:155px;top:5px;" tip="<%=rb.getString("YiChuSuoYou")%>">
                                <span class="el-icon-operation-clear el-icon"></span>
                            </div>
                            <div v-show='activeName == "batch"' @click="templateRowClick('all')" class="newIconBoxCls-bt" style="right:120px;top:5px;" tip="<%=rb.getString("Log")%>">
                                <span class="el-icon-operation-result el-icon"></span>
                            </div>
                            <div v-show='activeName == "batch" && hasConfigRole' :class="batchDeteCheck == true ? 'newIconBoxCls-bt' : 'newIconBoxCls-bt disabledCheck'" style="right:82px;top:5px;" tip="<%=rb.getString("ZiDongJianCeCanShu")%>" @click="DetectionConfig">
                                <span class="el-icon-circle-checkParam el-icon"></span>
                            </div>
                            <div v-if="hasConfigRole" class="newIconBoxCls-bt" style="right:46px;top:5px;" tip="<%=rb.getString("DaoRu")%>" @click="importFileBtn">
                                <span class="el-icon-operation-import el-icon"></span>
                            </div>
                            <el-popover trigger="click" placement="bottom-end" popper-class="monitorBtnPopperCls">
                                <div class="el-card__header">
                                    <%=rb.getString("DaoChu")%>
                                    <span style="color: 999;font-weight: normal;margin-left: 5px;">(<%=rb.getString("SuoYouCanShu")%>)</span>
                                    <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                                </div>
                                <div class="enbExportContent" style="width: 920px; max-height: 500px;overflow: auto;padding:20px 30px;">
                                    <div>
                                        <el-ctable ref="ctableDeviceGroup" height="350px" url="${ctx}/cell/fault/getDeviceGroup.action" @selection-change="deviceGroupSelect"
                                            :page-size="pageSize" pagination="true" :rownumber=true row-key="groupId" style="border-bottom:1px solid #E9E9E9;">
                                            <el-table-column type="selection"></el-table-column>
                                            <el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>'  prop="groupName" ></el-table-column>
                                        </el-ctable>
                                        <p style='color:#FA5555;margin:0px 0px 10px 5px;' v-show="showDeviceMsg"><%=rb.getString("QingXuanZeSheBeiZu")%></p>
                                    </div>
                                    <div v-if="false">
                                        <el-checkbox :indeterminate="!colAll" v-model="colAll" @change="colAllChange"></el-checkbox> 
                                        <span><%=rb.getString("QuanXuan")%></span>
                                    </div>
                                    <div>
                                        <div class="select-all-cls">
                                            <el-checkbox :indeterminate="form.device.length<deviceCol.length" v-model="basicAll" @change="deviceAllChange"></el-checkbox> 
                                            <span><%=rb.getString("QuanXuan")%></span>
                                        </div>
                                        <el-checkbox-group class="col-group" v-model="form.device">
                                            <el-checkbox v-for="item in deviceCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
                                        </el-checkbox-group>
                                    </div>
                                    <div v-if="false">
                                        <div class="select-all-cls">
                                            <el-checkbox v-model="cellAll" @change="cellAllChange"></el-checkbox> 
                                            <span><%=rb.getString("LinQu")%></span>
                                        </div>
                                    </div>
                                    <div v-if="false">
                                        <div class="select-all-cls">
                                            <el-checkbox v-model="freqAll" @change="freqAllChange"></el-checkbox> 
                                            <span><%=rb.getString("LinPin")%></span>
                                        </div>
                                    </div>
                                </div>
                                <div style="padding: 20px;">
                                    <el-button type="primary" @click="exportFileBtn"><%=rb.getString("QueDing")%></el-button>
                                    <el-button @click="closeExport"><%=rb.getString("QuXiao")%></el-button>
                                    <span style="color: 999;font-weight: normal;margin-left: 20px;"><i class="el-icon el-icon-circle-info gray-color"></i> <%=rb.getString("DengDaiTiShi")%></span>
                                </div>
                                <div v-show="activeName == 'batch'" slot="reference" class="newIconBoxCls-bt exportTipCls" style="right:10px;top:5px;" tip='<%=rb.getString("PiLiangPeiZhiDaoChuTiShi")%>'>
                                    <span class="el-icon-operation-export el-icon"></span>
                                </div>	
                            </el-popover>
                        </div>
                        <div>
                            <el-radio-group v-model='activeName' class="commonRadioButton" @change="tabClick" style='position: absolute; left: 20px; top: 46px;'>
                                <el-radio-button label="batch" class='batchRadio'><%=rb.getString("PiLiangPeiZhi")%></el-radio-button>
                                <el-radio-button label="neighborFrequencyConfig" class='neiRadio'><%=rb.getString("LTELinPinPeiZhi")%></el-radio-button>
                                <el-radio-button label="neighborCellConfig" class='cellRadio'><%=rb.getString("LTELinQuPeiZhi")%></el-radio-button>

                                <el-radio-button label="gsmNeighborCellConfig" class='neiRadio'><%=rb.getString("GSMLinQuPeiZhi")%></el-radio-button>
                                <el-radio-button label="gnbNeighborCellConfig" class='cellRadio'><%=rb.getString("5GLinQuPeiZhi")%></el-radio-button>
                            </el-radio-group>

                            <div v-show='activeName == "batch"'>
                                <el-ctable :id="'device_list'" :url='url' :time="6" ref='ctable' row-key="id" :query-params="params" height='calc(100% - 47px)' page-size="50" pagination="true" @selection-change='batchSelect' @load-success="deviceListLoadSuccess" class="commonTable">
                                    <template slot="toolbar">
                                        <div class='commonFlex' style='position: absolute; right: 15px; top: 46px;'>
                                            <div class='queryGroup'>
                                                <el-input @keyup.enter.native="query" v-model="configSearchText" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
                                                <i class="el-icon-common-search el-icon" @click='query' style="margin-left: 10px;"></i>
                                            </div>
                                            <div class='commonFlex' style='margin-left: 20px;'>
                                                <div class='commonSelectLabel'><%=rb.getString("ZhuangTai")%></div>
                                                <el-select v-model="statusParam" placeholder="<%=rb.getString("ZhuangTai")%>" @change="deviceListStatusChange">
                                                    <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
                                                </el-select>
                                            </div>
                                        </div>
                                    </template>
                                    <el-table-column v-if="hasConfigRole" type="selection" :reserve-selection="true" ></el-table-column>
                                    <el-table-column label='' width="80">
                                        <template slot-scope="scope">
                                            <div class=''>
                                                <span @click='templateRowClick(scope.row)' class='el-icon el-icon-operation-result' style='cursor: pointer;'></span>
                                                <span v-if="hasConfigRole" class="el-icon el-icon-main-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer; margin-left: 8px; margin-top: 6px;"></span>
                                            </div>
                                        </template>
                                    </el-table-column>							
                                    <el-table-column prop="connection_status" width="45">
                                        <template slot-scope="scope">
                                            <div :class="{
                                                'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
                                                '':scope.row.have_connected==2,
                                                'conn_exc':scope.row.connection_status=='Exception',
                                                'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
                                        </template>
                                    </el-table-column>

                                    <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" min-width="180" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("JianChaZhuangTai")%>' prop="detectionField" min-width="100" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField != undefined'>
                                                <span class='showHighlightColor'><%=rb.getString("BuYiZhi")%></span>
                                            </div>
                                            <div v-else-if='scope.row.detectionTime != null && scope.row.detectionTime != "" && scope.row.detectionTime != undefined'>
                                                <span class='hideHighlightColor'><%=rb.getString("YiZhi")%></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("FaSongZhuangTai")%>' prop="status" min-width="180" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <!-- 0:成功（已生效）；1：失败（生效失败）；2：进行中（生效中）；3：未生效 4:设备不在线 5：未执行-->				
                                            <div v-if="scope.row.status == '0'" class="statusTip">						
                                                <span class="el-icon el-icon-status-success  successStatus"></span>
                                                <span><%=rb.getString("YiShengXiao")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '1'" class="statusTip">						
                                                <span class="el-icon el-icon-status-failed  failStatus"></span>
                                                <span><%=rb.getString("ShengXiaoShiBai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '2'">						
                                                <span class="status_inProgress"></span>
                                                <span><%=rb.getString("ANRShengXiaoZhong")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '3'" class="statusTip">						
                                                <span class="el-icon el-icon-status-waiting1 wsxStatus "></span>
                                                <span><%=rb.getString("DengDai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '4'" class="statusTip">						
                                                <span class="el-icon el-icon-operation-restart wsxStatus "></span>
                                                <span><%=rb.getString("LiXianDaiZhiXing")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '5'" class="statusTip">						
                                                <span class="el-icon el-icon-status-notstarted" style="color: #666666;"></span>
                                                <span><%=rb.getString("WeiZhiXing")%></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("ZhiChiPinDuan")%>' prop="bands_support" min-width="140" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("bands_support") != -1'>
                                                <span v-if='scope.row.bands_support == ""' class='showHighlightColor'>null</span>
                                                <span v-else class='showHighlightColor'>{{scope.row.bands_support}}</span>
                                            </div>
                                            <div v-else>
                                                <span class='hideHighlightColor'>{{scope.row.bands_support}}</span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("DaiKuan")%>(MHz)' prop="band_width" min-width="150" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("band_width") != -1'>
                                                <span v-if='scope.row.band_width == ""' class='showHighlightColor'>null</span>
                                                <span v-else-if='scope.row.band_width == "n25"' class='showHighlightColor'>5</span>
                                                <span v-else-if='scope.row.band_width == "n50"' class='showHighlightColor'>10</span>
                                                <span v-else-if='scope.row.band_width == "n75"' class='showHighlightColor'>15</span>
                                                <span v-else-if='scope.row.band_width == "n100"' class='showHighlightColor'>20</span>
                                            </div>
                                            <div v-else>
                                                <span v-if='scope.row.band_width == "n25"' class='hideHighlightColor'>5</span>
                                                <span v-else-if='scope.row.band_width == "n50"' class='hideHighlightColor'>10</span>
                                                <span v-else-if='scope.row.band_width == "n75"' class='hideHighlightColor'>15</span>
                                                <span v-else-if='scope.row.band_width == "n100"' class='hideHighlightColor'>20</span>
                                            </div>
                                        </template>
                                    </el-table-column>									
                                    <el-table-column label='<%=rb.getString("PinDian")%>' prop="frequency" min-width="180" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("frequency") != -1'>
                                                <span v-if='scope.row.frequency == ""' class='showHighlightColor'>null</span>
                                                <span v-else v-html="earfcnFmt(scope.row,scope.row.frequency,scope.$index)" class='showHighlightColor'></span>
                                            </div>
                                            <div v-else>
                                                <span v-if='scope.row.frequency == ""'></span>
                                                <span v-else v-html="earfcnFmt(scope.row,scope.row.frequency,scope.$index)" class='hideHighlightColor'></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("ZiZhenPeiBi")%>' prop="subframe_assignment" min-width="200" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("subframe_assignment") != -1'>
                                                <span v-if='scope.row.subframe_assignment == ""' class='showHighlightColor'>null</span>
                                                <span v-else-if='scope.row.subframe_assignment == "0"' class='showHighlightColor'>0(DL:UL = 1:3)</span>
                                                <span v-else-if='scope.row.subframe_assignment == "1"' class='showHighlightColor'>1(DL:UL = 2:2)</span>
                                                <span v-else-if='scope.row.subframe_assignment == "2"' class='showHighlightColor'>2(DL:UL = 3:1)</span>
                                                <span v-else-if='scope.row.subframe_assignment == "6"' class='showHighlightColor'>6(DL:UL = 3:5)</span>
                                            </div>
                                            <div v-else>
                                                <span v-if='scope.row.subframe_assignment == "0"' class='hideHighlightColor'>0(DL:UL = 1:3)</span>
                                                <span v-else-if='scope.row.subframe_assignment == "1"' class='hideHighlightColor'>1(DL:UL = 2:2)</span>
                                                <span v-else-if='scope.row.subframe_assignment == "2"' class='hideHighlightColor'>2(DL:UL = 3:1)</span>
                                                <span v-else-if='scope.row.subframe_assignment == "6"' class='hideHighlightColor'>6(DL:UL = 3:5)</span>
                                            </div>
                                        </template>
                                    </el-table-column>											
                                    <el-table-column label='<%=rb.getString("TeShuZiZhenPeiBi")%>' prop="special_subframe_patterns" min-width="200" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("special_subframe_patterns") != -1'>
                                                <span v-if='scope.row.special_subframe_patterns == ""' class='showHighlightColor'>null</span>
                                                <span v-else class='showHighlightColor'>{{scope.row.special_subframe_patterns}}</span>
                                            </div>
                                            <div v-else>
                                                <span class='hideHighlightColor'>{{scope.row.special_subframe_patterns}}</span>
                                            </div>
                                        </template>
                                    </el-table-column>					
                                    <el-table-column label='<%=rb.getString("PLMN")%>' prop="plmn_id" min-width="100" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("plmn_id") != -1'>
                                                <span v-if='scope.row.plmn_id == ""' class='showHighlightColor'>null</span>
                                                <span v-else class='showHighlightColor'>{{scope.row.plmn_id}}</span>
                                            </div>
                                            <div v-else>
                                                <span class='hideHighlightColor'>{{scope.row.plmn_id}}</span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("TAC")%>' prop="tac" min-width="100" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("tac") != -1'>
                                                <span v-if='scope.row.tac == ""' class='showHighlightColor'>null</span>
                                                <span v-else class='showHighlightColor'>{{scope.row.tac}}</span>
                                            </div>
                                            <div v-else>
                                                <span class='hideHighlightColor'>{{scope.row.tac}}</span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("CPETxPower")%>' prop="txPower" min-width="120" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("txPower") != -1'>
                                                <span v-if='scope.row.txPower == ""' class='showHighlightColor'>null</span>
                                                <span v-else v-html="txPowerFmt(scope.row,scope.row.txPower,scope.$index)" class='showHighlightColor'></span>
                                            </div>
                                            <div v-else>
                                                <span v-html="txPowerFmt(scope.row,scope.row.txPower,scope.$index)" class='hideHighlightColor'></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='ECI' prop="cell_identity" min-width="100" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("cell_identity") != -1'>
                                                <span v-if='scope.row.cell_identity == ""' class='showHighlightColor'>null</span>
                                                <span v-else class='showHighlightColor'>{{scope.row.cell_identity}}</span>
                                            </div>
                                            <div v-else>
                                                <span class='hideHighlightColor'>{{scope.row.cell_identity}}</span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("PCI")%>' prop="phycellid" min-width="100" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("phycellid") != -1'>
                                                <span v-if='scope.row.phycellid == ""' class='showHighlightColor'>null</span>
                                                <span v-else class='showHighlightColor'>{{scope.row.phycellid}}</span>
                                            </div>
                                            <div v-else>
                                                <span class='hideHighlightColor'>{{scope.row.phycellid}}</span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("GenXuLieSuoYin")%>' prop="root_sequence_index" min-width="200" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("root_sequence_index") != -1'>
                                                <span v-if='scope.row.root_sequence_index == ""' class='showHighlightColor'>null</span>
                                                <span v-else class='showHighlightColor'>{{scope.row.root_sequence_index}}</span>
                                            </div>
                                            <div v-else>
                                                <span class='hideHighlightColor'>{{scope.row.root_sequence_index}}</span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='MME IP' prop="mme_ip" min-width="150" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.detectionField != null && scope.row.detectionField != "" && scope.row.detectionField.indexOf("mme_ip") != -1'>
                                                <span v-if='scope.row.mme_ip == ""' class='showHighlightColor'>null</span>
                                                <span else class='showHighlightColor'>{{scope.row.mme_ip}}</span>
                                            </div>
                                            <div v-else>
                                                <span class='hideHighlightColor'>{{scope.row.mme_ip}}</span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                </el-ctable>
                                <el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>
                            </div>
                            <!-- Neighbor Frequency configuration table   -->
                            <div v-show='activeName == "neighborFrequencyConfig"'>
                                <el-ctable 
                                    ref='neighborFrequencyTable' :time="6" row-key="id" class="commonTable"
                                    :id="'neighborFrequency_list'" 
                                    :url='neighborFrequencyUrl'
                                    :query-params="neighborFrequencyParams" 
                                    height='calc(100% - 47px)' 
                                    page-size="50" 
                                    pagination="true" @selection-change='batchSelect' @load-success="neighborFreLoadSuccess">
                                    <template slot="toolbar">
                                        <div class='queryGroup' style='position: absolute; right: 15px; top: 46px;'>
                                            <el-input @keyup.enter.native="neighborFrequencyQuery" v-model="neighborFrequencySearchText" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
                                            <i class="el-icon-common-search el-icon" @click='neighborFrequencyQuery' style="margin-left: 10px;"></i>
                                        </div>
                                    </template>
                                    <el-table-column v-if="hasConfigRole" type="selection" :reserve-selection="true" ></el-table-column>
                                    <el-table-column v-if="hasConfigRole" label='' width="30">
                                        <template slot-scope="scope">
                                            <div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status" min-width="180" show-overflow-tooltip> 
                                        <template slot-scope="scope">
                                            <!-- 0:成功（已生效）；1：失败（生效失败）；2：进行中（生效中）；3：未生效 -->				
                                            <div v-if="scope.row.status == '0'" class="statusTip">						
                                                <span class="el-icon el-icon-status-success  successStatus"></span>
                                                <span><%=rb.getString("YiShengXiao")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '1'" class="statusTip">						
                                                <span class="el-icon el-icon-status-failed  failStatus"></span>
                                                <span><%=rb.getString("ShengXiaoShiBai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '2'">						
                                                <span class="status_inProgress"></span>
                                                <span><%=rb.getString("ANRShengXiaoZhong")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '3'" class="statusTip">						
                                                <span class="el-icon el-icon-status-waiting1 wsxStatus "></span>
                                                <span><%=rb.getString("DengDai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '4'" class="statusTip">						
                                                <span class="el-icon el-icon-operation-restart wsxStatus "></span>
                                                <span><%=rb.getString("LiXianDaiZhiXing")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '5'" class="statusTip">						
                                                <span class="el-icon el-icon-status-notstarted" style="color: #666666;"></span>
                                                <span><%=rb.getString("WeiZhiXing")%></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason" min-width="180" show-overflow-tooltip></el-table-column>					
                                    <el-table-column label='<%=rb.getString("WanChengShiJian")%>' prop="exeTime" min-width="150" show-overflow-tooltip></el-table-column>					
                                    <el-table-column label='<%=rb.getString("PinDian")%>' prop="earfcn" :formatter='frequencyFmt' min-width="200" show-overflow-tooltip></el-table-column>				
                                    <el-table-column label='Q-OffsetRange' prop="qOffsetRange" min-width="200" show-overflow-tooltip></el-table-column>							
                                    <el-table-column label='Q-RxLevMin' prop="qRxLevMin" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoQuZhongXuanYouXianJi")%>' prop="reselectionPriority" min-width="200" show-overflow-tooltip></el-table-column>				
                                    <el-table-column label='<%=rb.getString("GaoChongXuanMenXian")%>' prop="reselectionThreshHigh" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("DiChongXuanMenXian")%>' prop="reselectionThreshLow" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("UEZuiDaFaSongGongLv")%>' prop="pMax" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("ChongXuanDingShiQi")%>' prop="tReselectionEutra" min-width="200" show-overflow-tooltip></el-table-column>	
                                </el-ctable>
                                <el-cmenu ref="neighborFrequencyMenu" :data="menus" @click="clickMenu"></el-cmenu>
                            </div>
                            <!--LTE 邻区 configuration table -->
                            <div v-show='activeName == "neighborCellConfig"'>
                                <el-ctable 
                                    ref='neighborCellTable' :time="6" row-key="id" class="commonTable"
                                    :id="'neighborCell_list'" 
                                    :url='neighborCellUrl'
                                    :query-params="neighborCellParams" 
                                    height='calc(100% - 47px)' 
                                    page-size="50" 
                                    pagination="true"
                                    @selection-change='batchSelect' @load-success="neighborCellLoadSuccess">
                                    <template slot="toolbar">
                                        <div class='queryGroup' style='position: absolute; right: 15px; top: 46px;'>
                                            <el-input @keyup.enter.native="neighborCellQuery" v-model="neighborCellSearchText" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
                                            <i class="el-icon-common-search el-icon" @click='neighborCellQuery' style="margin-left: 10px;"></i>
                                        </div>
                                    </template>
                                    <el-table-column v-if="hasConfigRole" type="selection" :reserve-selection="true" ></el-table-column>
                                    <el-table-column v-if="hasConfigRole" label='' width="30">
                                        <template slot-scope="scope">
                                            <div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status" min-width="180" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <!-- 0:成功（已生效）；1：失败（生效失败）；2：进行中（生效中）；3：未生效 -->				
                                            <div v-if="scope.row.status == '0'" class="statusTip">						
                                                <span class="el-icon el-icon-status-success  successStatus"></span>
                                                <span><%=rb.getString("YiShengXiao")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '1'" class="statusTip">						
                                                <span class="el-icon el-icon-status-failed  failStatus"></span>
                                                <span><%=rb.getString("ShengXiaoShiBai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '2'">						
                                                <span class="status_inProgress"></span>
                                                <span><%=rb.getString("ANRShengXiaoZhong")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '3'" class="statusTip">						
                                                <span class="el-icon el-icon-status-waiting1 wsxStatus "></span>
                                                <span><%=rb.getString("DengDai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '4'" class="statusTip">						
                                                <span class="el-icon el-icon-operation-restart wsxStatus "></span>
                                                <span><%=rb.getString("LiXianDaiZhiXing")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '5'" class="statusTip">						
                                                <span class="el-icon el-icon-status-notstarted" style="color: #666666;"></span>
                                                <span><%=rb.getString("WeiZhiXing")%></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason" min- width="260" show-overflow-tooltip></el-table-column>					
                                    <el-table-column label='<%=rb.getString("WanChengShiJian")%>' prop="exeTime" min-width="150" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoQuBianHao")%>' prop="cellIndex" min-width="140" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.cellIndex == "1"'>Cell 1</div>
                                            <div v-else-if='scope.row.cellIndex == "2"'>Cell 2</div>
                                        </template>
                                    </el-table-column>													
                                    <el-table-column label='<%=rb.getString("PinDian")%>' prop="earfcn" :formatter='frequencyFmt' min-width="180" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("PCI")%>' prop="pci" min-width="100" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoQuTeDingPianYiLiang")%>' prop="qOffset" min-width="120" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoQuDuLiPianYiLiang")%>' prop="cio" min-width="130" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("TAC")%>' prop="tac" min-width="120" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("PLMN")%>' prop="plmn" min-width="140" show-overflow-tooltip></el-table-column>
                                    
                                    <el-table-column label='ECI' prop="cellId" min-width="140" show-overflow-tooltip></el-table-column>
									<el-table-column label='eNodeB Type' prop="eNodeBType" min-width="140" show-overflow-tooltip>
										<template slot-scope="scope">
											<div v-if='scope.row.eNodeBType == "0"'>Macro</div>
											<div v-else-if='scope.row.eNodeBType == "1"'>Home</div>
										</template>
									</el-table-column>	

                                </el-ctable>
                                <el-cmenu ref="neighborCellMenu" :data="menus" @click="clickMenu"></el-cmenu>
                            </div>

                            <!--GSM 邻区 :url='gsmNeighborCellUrl'-->
                            <div v-show='activeName == "gsmNeighborCellConfig"'>
                                <el-ctable 
                                    ref='gsmNeighborCellTable' :time="6" row-key="id" class="commonTable"
                                    :id="'gsmNeighborCell_list'" 
                                    :url='gsmNeighborCellUrl'
                                    :query-params="gsmNeighborCellParams" 
                                    height='calc(100% - 47px)' 
                                    page-size="50" 
                                    pagination="true"
                                    @selection-change='batchSelect' @load-success="gsmNeighborCellLoadSuccess">
                                    <template slot="toolbar">
                                        <div class='queryGroup' style='position: absolute; right: 15px; top: 46px;'>
                                            <el-input @keyup.enter.native="gsmNeighborCellQuery" v-model="gsmNeighborCellSearchText" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
                                            <i class="el-icon-common-search el-icon" @click='gsmNeighborCellQuery' style="margin-left: 10px;"></i>
                                        </div>
                                    </template>
                                    <el-table-column v-if="hasConfigRole" type="selection" :reserve-selection="true" ></el-table-column>
                                    <el-table-column v-if="hasConfigRole" label='' width="30">
                                        <template slot-scope="scope">
                                            <div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
                                        </template>
                                    </el-table-column>
            
                                    <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status" min-width="180" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <!-- 0:成功（已生效）；1：失败（生效失败）；2：进行中（生效中）；3：未生效 -->				
                                            <div v-if="scope.row.status == '0'" class="statusTip">						
                                                <span class="el-icon el-icon-status-success  successStatus"></span>
                                                <span><%=rb.getString("YiShengXiao")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '1'" class="statusTip">						
                                                <span class="el-icon el-icon-status-failed  failStatus"></span>
                                                <span><%=rb.getString("ShengXiaoShiBai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '2'">						
                                                <span class="status_inProgress"></span>
                                                <span><%=rb.getString("ANRShengXiaoZhong")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '3'" class="statusTip">						
                                                <span class="el-icon el-icon-status-waiting1 wsxStatus "></span>
                                                <span><%=rb.getString("DengDai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '4'" class="statusTip">						
                                                <span class="el-icon el-icon-operation-restart wsxStatus "></span>
                                                <span><%=rb.getString("LiXianDaiZhiXing")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '5'" class="statusTip">						
                                                <span class="el-icon el-icon-status-notstarted" style="color: #666666;"></span>
                                                <span><%=rb.getString("WeiZhiXing")%></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason" min-width="260" show-overflow-tooltip></el-table-column>					
                                    <el-table-column label='<%=rb.getString("WanChengShiJian")%>' prop="exeTime" min-width="150" show-overflow-tooltip></el-table-column>	
                                    <el-table-column label='<%=rb.getString("XiaoQuBianHao")%>' prop="cellIndex" min-width="160" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.cellIndex == 1'>Cell 1</div>
                                            <div v-else-if='scope.row.cellIndex == 2'>Cell 2</div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='ARFCN' prop="arfcn" min-width="180" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("PLMN")%>' prop="plmn" min-width="140" show-overflow-tooltip></el-table-column>

                                    <el-table-column label='<%=rb.getString("WeiZhiQuXinXi")%>' prop="lac" min-width="210" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("JiZhanShiBieMa")%>' prop="bsic" min-width="160" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("XIAOQUID")%>' prop="cellId" min-width="140" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("JiHuoZhuangTai")%>' prop="activeState" min-width="120" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.activeState == "true"'><%=rb.getString("JiHuo")%></div>
                                            <div v-else-if='scope.row.activeState == "false"'><%=rb.getString("QuJiHuo")%></div>
                                        </template>
                                    </el-table-column>
                                </el-ctable>
                                <el-cmenu ref="gsmNeighborCellMenu" :data="menus" @click="clickMenu"></el-cmenu>
                            </div>

                            <!--5g 邻区 configuration table :url='gnbNeighborCellUrl'-->
                            <div v-show='activeName == "gnbNeighborCellConfig"'>
                                <el-ctable 
                                    ref='gnbNeighborCellTable' :time="6" row-key="id" class="commonTable"
                                    :id="'gnbNeighborCell_list'" 
                                    :url='gnbNeighborCellUrl'
                                    :query-params="gnbNeighborCellParams" 
                                    height='calc(100% - 47px)' 
                                    page-size="50" 
                                    pagination="true"
                                    @selection-change='batchSelect' @load-success="gnbNeighborCellLoadSuccess">
                                    <template slot="toolbar">
                                        <div class='queryGroup' style='position: absolute; right: 15px; top: 46px;'>
                                            <el-input @keyup.enter.native="gnbNeighborCellQuery" v-model="gnbNeighborCellSearchText" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
                                            <i class="el-icon-common-search el-icon" @click='gnbNeighborCellQuery' style="margin-left: 10px;"></i>
                                        </div>
                                    </template>
                                    <el-table-column v-if="hasConfigRole" type="selection" :reserve-selection="true"></el-table-column>
                                    <el-table-column v-if="hasConfigRole" width="30">
                                        <template slot-scope="scope">
                                            <div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" min-width="200" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status" min-width="180" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <!-- 0:成功（已生效）；1：失败（生效失败）；2：进行中（生效中）；3：未生效 -->				
                                            <div v-if="scope.row.status == '0'" class="statusTip">						
                                                <span class="el-icon el-icon-status-success  successStatus"></span>
                                                <span><%=rb.getString("YiShengXiao")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '1'" class="statusTip">						
                                                <span class="el-icon el-icon-status-failed  failStatus"></span>
                                                <span><%=rb.getString("ShengXiaoShiBai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '2'">						
                                                <span class="status_inProgress"></span>
                                                <span><%=rb.getString("ANRShengXiaoZhong")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '3'" class="statusTip">						
                                                <span class="el-icon el-icon-status-waiting1 wsxStatus "></span>
                                                <span><%=rb.getString("DengDai")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '4'" class="statusTip">						
                                                <span class="el-icon el-icon-operation-restart wsxStatus "></span>
                                                <span><%=rb.getString("LiXianDaiZhiXing")%></span>
                                            </div>
                                            <div v-if="scope.row.status == '5'" class="statusTip">						
                                                <span class="el-icon el-icon-status-notstarted" style="color: #666666;"></span>
                                                <span><%=rb.getString("WeiZhiXing")%></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason" min-width="260" show-overflow-tooltip></el-table-column>					
                                    <el-table-column label='<%=rb.getString("WanChengShiJian")%>' prop="exeTime" min-width="150" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("XiaoQuBianHao")%>' prop="cellIndex" min-width="160" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if='scope.row.cellIndex == 1'>Cell 1</div>
                                            <div v-else-if='scope.row.cellIndex == 2'>Cell 2</div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("PLMN")%>' prop="plmn" min-width="140" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='SSB' prop="ssb" min-width="140" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("GNBBiaoShi")%>' prop="gnbId" min-width="120" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("GNBBiaoShiChangDu")%>' prop="gnbIdLength" min-width="150" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("XIAOQUID")%>' prop="cellId" min-width="140" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("PCI")%>' prop="pci"min- width="120" show-overflow-tooltip></el-table-column>

                                    <el-table-column label='QOFFSET' prop="qoffset" min-width="120" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='<%=rb.getString("TAC")%>' prop="tac" min-width="120" show-overflow-tooltip></el-table-column>
                                </el-ctable>
                                <el-cmenu ref="gnbNeighborCellMenu" :data="menus" @click="clickMenu"></el-cmenu>
                            </div>
                        </div>	
                    </div>
                    
                    <div :class="logType == 'all' ? 'taskListAllWarp':'taskListWarp'" v-show='showLog'>	
                        <el-ctable :id="'resultLists'" :url='resultUrl' :time="6" ref='resultCtable' row-key="id" :query-params="resultParams" style="height:100%; " class='newTabs commonWarp'
                            page-size="50" pagination="true">
                            <template slot="toolbar">
                                <div style=' height: 36px; margin-top: -10px;line-height: 36px;justify-content: space-between; padding: 0 20px;border-bottom: 1px solid #D5DCEC;'>
                                    <span class='commonText14'>{{logTitle}}</span>
                                    <span v-if="logType != 'all'" style='font-size: 14px; color: #4D84FF; font-weight: bold; margin: 4px 0px 0;'>({{selectSN}})</span>
                                    <div v-if="logType == 'all' && hasConfigRole" @click='clearLogs' class="kpiQueryExportClass newIconBoxCls-bt" style="top:5px; right: 100px;" >
                                        <span class="el-icon el-icon-operation-clear" ></span>
                                        <div class="titleButtonText"><%=rb.getString("QingKong")%></div>
                                    </div>
                                    <div class="kpiQueryExportClass newIconBoxCls-bt" style="top:5px; right: 60px;" @click='resultExport'>
                                        <span class="el-icon el-icon-operation-export" ></span>
                                        <div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
                                    </div>
                                    <div class="kpiQueryExportClass newIconBoxCls-bt" style="top:5px; right: 20px;" @click='showLog = false'>
                                        <span class="el-icon el-icon-close" ></span>
                                        <div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
                                    </div>
                                </div>
                                <div class='commonFlex' style='padding-top: 10px;'>
                                    <div class='queryGroup'>
                                        <el-input @keyup.enter.native="resultQuery" v-model="resultText" class='pairgrid-query' placeholder='<%=rb.getString("JianChaJieGuo")%>'></el-input>
                                        <i class="el-icon-common-search el-icon" @click='resultQuery' style="margin-left: 10px;"></i>
                                    </div>
                                    <div class='commonFlex' style='margin-left: 20px;'>
                                        <div class='commonSelectLabel'><%=rb.getString("FaSongZhuangTai")%></div>
                                        <el-select v-model="resultStatus" placeholder="<%=rb.getString("FaSongZhuangTai")%>" @change="resultStatusChange">
                                            <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
                                        </el-select>
                                    </div>
                                    <div class='commonFlex' style='margin-left: 20px;'>
                                        <div class='commonSelectLabel'><%=rb.getString("ChuLiCeLue")%></div>
                                        <el-select v-model="resultPolicy" placeholder="<%=rb.getString("ZhuangTai")%>" @change="resultPolicyChange">
                                            <el-option v-for="item in policyOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
                                        </el-select>
                                    </div>
                                </div>
                            </template>
                            <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber" show-overflow-tooltip></el-table-column>
                            <el-table-column label='<%=rb.getString("XuanZeZhiXingFangShi")%>' prop="executeType" show-overflow-tooltip>
                                <template slot-scope="scope">
                                    <div v-if="scope.row.executeType == 'automatic'" class="statusTip">						
                                        <span><%=rb.getString("eNBZiDongZhiXing")%></span>
                                    </div>
                                    <div v-else class="statusTip">						
                                        <span><%=rb.getString("eNBShouDongZhiXing")%></span>
                                    </div> 
                                </template>
                            </el-table-column>

                            <el-table-column label='<%=rb.getString("FaSongZhuangTai")%>' prop="status" show-overflow-tooltip>
                                <template slot-scope="scope">
                                    <!-- 0:成功（已生效）；1：失败（生效失败）；2：进行中（生效中）；3：未生效 -->			
                                    <div v-if="scope.row.status == '0'" class="statusTip">						
                                        <span class="el-icon el-icon-status-success  successStatus"></span>
                                        <span><%=rb.getString("YiShengXiao")%></span>
                                    </div>
                                    <div v-if="scope.row.status == '1'" class="statusTip">						
                                        <span class="el-icon el-icon-status-failed  failStatus"></span>
                                        <span><%=rb.getString("ShengXiaoShiBai")%></span>
                                    </div>
                                    <div v-if="scope.row.status == '2'">						
                                        <span class="status_inProgress"></span>
                                        <span><%=rb.getString("ANRShengXiaoZhong")%></span>
                                    </div>
                                    <div v-if="scope.row.status == '3'" class="statusTip">						
                                        <span class="el-icon el-icon-status-waiting1 wsxStatus "></span>
                                        <span><%=rb.getString("DengDai")%></span>
                                    </div>
                                    <div v-if="scope.row.status == '4'" class="statusTip">						
                                        <span class="el-icon el-icon-operation-restart wsxStatus "></span>
                                        <span><%=rb.getString("LiXianDaiZhiXing")%></span>
                                    </div>
                                    <div v-if="scope.row.status == '5'" class="statusTip">						
                                        <span class="el-icon el-icon-status-notstarted" style="color: #666666;"></span>
                                        <span><%=rb.getString("WeiZhiXing")%></span>
                                    </div>
                                </template>
                            </el-table-column>
                            <el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason" show-overflow-tooltip></el-table-column>					
                            <el-table-column label='<%=rb.getString("JianChaJieGuo")%>' prop="detectionResult" show-overflow-tooltip></el-table-column>					
                            <el-table-column label='<%=rb.getString("ChuLiCeLue")%>' prop="processStrategy" show-overflow-tooltip>
                                <template slot-scope="scope">
                                    <!-- 1: 已生效  0： 生效失败-->	
                                    <div v-if="scope.row.processStrategy == 'prompt'">						
                                        <span><%=rb.getString("TiShiOption")%></span>
                                    </div> 			
                                    <div v-else-if="scope.row.processStrategy == 'promptAndModify'">						
                                        <span><%=rb.getString("TiShiBingZiDongGaiZheng")%></span>
                                    </div>
                                </template>
                            </el-table-column>
                            <el-table-column v-if="false" label='<%=rb.getString("ChongShiCeLue")%>' prop="retryStrategy" show-overflow-tooltip></el-table-column>					
                            <el-table-column label='<%=rb.getString("JianCeShiJian")%>' prop="detectionTime" show-overflow-tooltip></el-table-column>					
                            <el-table-column label='<%=rb.getString("PeiZhiShiJian")%>' prop="exeTime" show-overflow-tooltip></el-table-column>
                        </el-ctable>
                    </div>
                </div>
                <div class='rightWarp' style='flex: 0 1 380px;' v-show='autoDetectionShow'>
                    <div class='rightWarpLayer'>
                        <div class='rightBoxHeaderHasTip'>
                            <div class='headerText'>
                                <span><%=rb.getString("ZiDongJianCe")%></span>
                                <span class='closeIconBox' @click='automicCancel'><i class='el-icon el-icon-close'></i></span>
                            </div>
                        </div>
                        <div class='rightWarpLayerContent'>
                            <el-form ref="detectionForm" label-position="top" :model="detectionForm" :rules="detectionRules" class='reportFormBox' style='margin: 20px 20px;'>
                                <el-form-item label='<%=rb.getString("SheZhiKaiGuan") %>' prop='enable' class='commonFlex switchItem'>
                                    <el-switch v-model="detectionForm.enable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
                                </el-form-item>
                                <el-form-item label='<%=rb.getString("SheBeiXuanZe")%>' style='margin-bottom: 0;' v-if='false'>
                                    <el-ctable 
                                        ref='selectDevices' row-key="serial_number"
                                        :id="'selectDevice'" 
                                        :url='selectDevicesUrl' 
                                        :query-params="selectDeviceParams" 
                                        height='260px' 
                                        :default-checked="defaultCheckedGroup"
                                        :show-pager="false" pagination="true" :rownumber="false" style='border: 1px solid #D5DCEC; width: 326px; margin: 0;'
                                        @selection-change='selectDevicesChange'>
                                        <template slot="toolbar">
                                            <div class='queryGroup'>
                                                <el-input @keyup.enter.native="selectDeviceQuery" v-model="selectDeviceSearchText" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
                                                <i class="el-icon-common-search el-icon" @click='selectDeviceQuery' style="margin-left: 10px;"></i>
                                            </div>
                                        </template>
                                        <el-table-column type="selection" :reserve-selection="true" ></el-table-column>
                                        <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
                                </el-form-item>
                                <el-form-item label=" <%=rb.getString("JianCeShiJian")%>" prop="time">
                                    <div class='commonFlex'>
                                        <el-select v-model="detectionForm.time" placeholder="<%=rb.getString("JianCeShiJian")%>" style='width: 230px; margin-right: 16px;'>
                                            <el-option v-for="item in hourArr" :label="item.label" :value="item.value"></el-option>
                                        </el-select>
                                        <div>
                                            <el-checkbox v-model='detectionForm.isEveryDay' true-label="true" false-label="false" @change=""></el-checkbox>
                                            <span class='commonGeneral12' style='margin-left: 6px;'> <%=rb.getString("MeiTian")%></span>
                                        </div>
                                    </div>
                                </el-form-item>
                                <el-form-item label='<%=rb.getString("BuYiZhiCeLue")%>' prop='processStrategy' style='margin-bottom: 10px;'>
                                    <el-radio-group v-model="detectionForm.processStrategy">
                                        <el-radio label="prompt" border><%=rb.getString("TiShiOption")%></el-radio>
                                        <el-radio label="promptAndModify" border style='margin-left: 20px;'><%=rb.getString("TiShiBingZiDongGaiZheng")%></el-radio>
                                    </el-radio-group>
                                    
                                </el-form-item>
                                <!-- 选中此项，要做参数一致性检查， 如果不一致，则参数下发后才会重启， 都一致，不用再次下发参数，不用重启 -->
                                <div v-show='detectionForm.processStrategy == "promptAndModify"' class='commonNotes12' style='margin-bottom: 20px;'><%=rb.getString("enbHuiBeiChongQi")%></div>
                                <el-form-item v-show='detectionForm.processStrategy == "promptAndModify"' label="<%=rb.getString("XiuGaiChongShiCeLue")%>" prop="retyStrategy" placeholder="<%=rb.getString("YouXiangShuRuYaoQiu")%>" class='retryInput' style='margin-bottom: 15px; '>
                                <el-input v-model="detectionForm.retyStrategy" >
                                        <template slot='append'><%=rb.getString("ChongShiCiShu")%></template>   
                                </el-input>
                                </el-form-item>
                            </el-form>
                        </div>
                        <div class='commonFlex commonBorderTop commonFormFotter'>
                            <div>
                                <el-button type="primary" @click="automicSubmit"><%=rb.getString("QueDing")%></el-button>
                                <el-button @click="automicCancel"><%=rb.getString("QuXiao")%></el-button>
                            </div>
                        </div>
                    </div>
                </div>
                <!-- 导入 -->
                <div class='rightWarp rightImportBox' style='flex: 0 1 360px;' v-show='showImportCard'>
                    <div class='rightWarpLayer'>
                        <div class='rightBoxHeaderHasTip'>
                            <div class='headerText'>
                                <span><%=rb.getString("DaoRu")%></span>
                                <span class='closeIconBox' @click='closeImportDevice'><i class='el-icon el-icon-close'></i></span>
                            </div>
                        </div>
                        <div class='rightWarpLayerContent'>
                            <el-form label-position="top" ref="importRuleForm" :model='importRuleForm' :rules='importRules' style='margin: 20px;'>     		     			            
                                <el-form-item label="<%=rb.getString("DaoRuLeiXing")%>">
                                    <el-radio-group v-model="importRuleForm.importType">
                                        <el-radio label="append" border><%=rb.getString("ZhuiJia")%></el-radio>
                                        <el-radio label="cover" border><%=rb.getString("TiHuan")%></el-radio>
                                    </el-radio-group>
                                </el-form-item>
                                <el-form-item prop="fileName">
                                    <template slot='label'>
                                        <div class='commonFlex'>
                                            <span class='commonSize14'><%=rb.getString("DaoRuWenJian")%></span>
                                            <span class='commonNotes12'> (<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>)</span>
                                        </div>
                                    </template>
                                    <el-upload 
                                        ref="upload"
                                        :before-upload='beforeUpload' 
                                        :on-success='checkFile' 
                                        :on-change="fileChange"  
                                        :show-file-list=false 	                  		
                                        :action="importRuleForm.uploadFileUrl" 
                                        :data="fileParams" 
                                        name="uploadFile" 
                                        accept=".xls,.xlsx"
                                        :auto-upload="false">
                                        <el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
                                            <a slot="append" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
                                        </el-input>								
                                        <a slot="trigger" ref="file_up"></a>
                                    </el-upload>	
                                </el-form-item> 
                                <el-form-item label="">
                                    <div>
                                        <span class='commonNotes12' style='line-height: 22px;'><%=rb.getString("DaoRuWenJianTiShi")%></span>
                                        <span style="cursor:pointer;margin-left:10px;" @click="exportTemplate">
                                            <span class='el-icon el-icon-common-download'></span>
                                            <span class='commonGeneral12' style='text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
                                        </span>
                                    </div> 
                                </el-form-item> 
                                
                                <div style="width: 320px;line-height: 22px;" v-show='activeName == "gsmNeighborCellConfig" && importRuleForm.importType == "append"'>
                                    <span class="el-icon el-icon-circle-info infoTip"></span>
                                    <span class="commonNotes12"><%=rb.getString("WeiYiXingShuoMing")%></span><br/>
                                    <span class="commonNotes12" style="word-wrap: break-word;"><%=rb.getString("GSMWeiYiXingShuoMingNeiRong")%></span>
                                </div>
                                <div style="width: 320px;line-height: 22px;" v-show='activeName == "gnbNeighborCellConfig" && importRuleForm.importType == "append"'>
                                    <span class="el-icon el-icon-circle-info infoTip"></span>
                                    <span class="commonNotes12"><%=rb.getString("WeiYiXingShuoMing")%></span><br/>
                                    <span class="commonNotes12" style="word-wrap: break-word;"><%=rb.getString("GNBWeiYiXingShuoMingNeiRong")%></span>
                                </div>
                            </el-form>
                        </div>
                        <div class='commonFlex commonBorderTop commonFormFotter'>
                            <div>
                                <el-button type="primary" @click="uploadDevice" :disabled='importLoading'><%=rb.getString("QueDing")%></el-button>
                                <el-button @click="closeImportDevice"><%=rb.getString("QuXiao")%></el-button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </el-tab-pane>
        <el-tab-pane v-if="gsmEnable" label='GSM' name="GSM">
            <div id="gsmConfiguration" class="flex-item-cls" style="min-width: 1060px;"></div>
        </el-tab-pane>
    </el-tabs>
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :position="slidePosition" class='commonWarp'
		 :height="slideHeight" :modal='modal' :width="slideWidth" @ok='saveConfig' @cancel='cancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>	
	
	<!-- 下发-->
	<el-dialog title="<%=rb.getString("QueRen")%>" width="600px" :visible="showIssueCard" class="issueCard" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeIssueConfirm">
		<el-form label-position="top" ref="issueRuleForm" :model='issueRuleForm'>
         	<div v-if="activeAll == false"><%=rb.getString("QueRenKaiShiSuoXuanPeiZhi")%></div>
         	<div v-if="activeAll == true"><%=rb.getString("QueRenKaiShiSuoYouPeiZhi")%></div>
         	<div v-if='activeName == "batch"'>
         	    <div style="margin:10px 0;"><%=rb.getString("PeiZhiJiangHuiChongQiSheBei")%></div>
			    <el-checkbox v-model="issueRuleForm.isReboot" :true-label="1" :false-label="0" ><%=rb.getString("SheZhiHouChongQi")%></el-checkbox>
			</div>
        </el-form>
       	<div slot="footer">
          	 <el-button type="primary" @click="issueConfirm"><%=rb.getString("QueDing")%></el-button>
             <el-button @click="closeIssueConfirm"><%=rb.getString("QuXiao")%></el-button>
        </div>			
	</el-dialog>
</div>

<script type="text/javascript">
var updateTimer,suspendTimer;
var batchVue = new Vue({
	el:'#configurationPlan',
	data(){
		var vm = this,
    	validateFileNames = function(rule,value,callback) {
        	var value = vm.fileName; 
        	var pathSplit = value.split(/\\/);
		    var filename = pathSplit[pathSplit.length - 1];

			if( value === '' || value === null || value === undefined) {
				callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
			}else if(filename.length > 100){
				callback(new Error('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>'))
			}else if(!fileFormatMatch(value,"xlsx,xls")){
                callback(new Error("<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>"))
            }else {
				callback();
			} 
		},

		validateTime = (rule,value,callback) => {
			if(vm.detectionForm.enable == 'true'){
				if(value === "" || value === null || value === undefined){
					callback('<%=rb.getString("QingXuanZeShiJian")%>')
				}else{
					callback()
				}
			}else{
				callback();
			}
		},
		validateRetyStrategy = (rule,value,callback) => {
			var reg = /^[0-9]*$/;
			
			if(vm.detectionForm.enable == 'true' && vm.detectionForm.processStrategy == 'promptAndModify'){
				if(value === "" || value === null || value === undefined){
					callback('请输入次数')
				}else if(!reg.test(value) || (value < 1 || value > 3)){
					callback('<%=rb.getString("FanWei")%>: 1 ~ 3')
				}else{
					callback()
				}
			}else{
				if(!reg.test(value) || (value < 1 || value > 3)){
					callback('<%=rb.getString("FanWei")%>: 1 ~ 3')
				}else{
					callback()
				}
			}
		};
		var deviceCol= [
			{code: 'serial_number', label: '<%=rb.getString("XiaoZhanBianMa")%>',disabled: true},
			{code: 'host_name', label: '<%=rb.getString("HostName")%>'},
			{code: 'band', label: '<%=rb.getString("PinDuan")%>'},
			{code: 'bandwidth', label: '<%=rb.getString("DaiKuan")%>'},
			{code: 'EARFCNDLINUSE', label: '<%=rb.getString("PinDian")%>'},
			{code: 'plmnid', label: '<%=rb.getString("PLMN")%>'},
			{code: 'tac', label: '<%=rb.getString("TAC")%>'},
			{code: 'tx_power',label:'<%=rb.getString("CPETxPower")%>'},
			{code: 'CELL_IDENTITY', label: 'ECI'},
			{code: 'PHYCELLID', label: '<%=rb.getString("PCI2")%>'},
			{code: 'rootIndex', label: '<%=rb.getString("GenXuLieSuoYin")%>'},
			{code: 'mme_status', label: '<%=rb.getString("RSMME")%>'},
			{code: 'specialSubframe', label: '<%=rb.getString("TeShuZiZhenPeiBi")%>'},
			{code: 'signment', label: '<%=rb.getString("ZiZhenPeiBi")%>'},
		],
		ignores=[];
		return {
			activeName:"batch",
			url:'${ctx}/task/BatchConfiguration/getBatchConfigurationTaskPageList.action',
			menus:[],
			rowData:[],
			configSearchText:'',
			params:{
				timeZone:timeZone,
				searchText:'',
				status: ''
			},
						
			//lte 邻区配置
			neighborCellUrl:'${ctx}/task/BatchConfiguration/getBatchNeighbourCellInfoPageList.action',
			neighborCellSearchText:'',
			neighborCellParams:{
				timeZone:timeZone,
				searchText:''
			},
			
			//临频配置
			neighborFrequencyUrl:'${ctx}/task/BatchConfiguration/getBatchNeighbourFreqInfoPageList.action',
			neighborFrequencySearchText:'',
			neighborFrequencyParams:{
				timeZone:timeZone,
				searchText:''
			},

			//GSM 邻区配置
			gsmNeighborCellUrl:'${ctx}/task/BatchConfiguration/queryBatchGSMNcellTaskPageList.action',
			curColumn: {},
			gsmNeighborCellSearchText:'',
			gsmNeighborCellParams:{
				timeZone: timeZone,
				searchText:'',
				status: ''
			},
			//5G 邻区配置
			gnbNeighborCellUrl:'${ctx}/task/BatchConfiguration/queryBatchGNBNcellTaskPageList.action',
			gnbNeighborCellSearchText:'',
			gnbNeighborCellParams:{
				timeZone: timeZone,
				searchText:'',
				status: ''
			},
	
			//slide
			slideUrl:'',
			slideTitle:'',
			slideFooter:false,
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
			modal:false,
			
			//import
			showImportCard:false,
			importRuleForm: {
	            uploadFileUrl: '',
	            importType: "append",
	       	},	         	          
	        fileParams:{},              
	        fileName:'',	            					
			showFileTip:false,
			fileList:[],
			filePath:'',
			importRules: {	           		
				fileName:[
	            	{ validator: validateFileNames},
	            ]
	        },
	        
	        //下发确认
	        showIssueCard:false,
	        issueRuleForm: {
	        	isReboot: '0'
	       	},

			//下发，默认显示，不置灰
			issueBtnShow:true,
			issueDisabled:false,
			//终止，默认隐藏，不置灰(false)
			suspendBtnShow:false,
			suspendDisabled:false,
			defaultIssueBtnShow:true,
			batchDeteCheck: false,
			defaultIssueDisabled:true,
			basicTotal:'',
			neighborCellTotal:'',
			neighborFreTotal:'',
			gsmNeighborCellTotal:'',
			gnbNeighborCellTotal:'',
			paramTotal: '',
			selectionData:[],		
			importLoading: false,
			
			selectSN: '',
			statusOptions: [
				{'label': '<%=rb.getString("QuanBu")%>', 'value': ''},
				{'label': '<%=rb.getString("YiShengXiao")%>', 'value': '0'},
				{'label': '<%=rb.getString("ShengXiaoShiBai")%>', 'value': '1'},
				{'label': '<%=rb.getString("ANRShengXiaoZhong")%>', 'value': '2'},
				{'label': '<%=rb.getString("DengDai")%>', 'value': '3'},
				{'label': '<%=rb.getString("LiXianDaiZhiXing")%>', 'value': '4'},
				{'label': '<%=rb.getString("WeiZhiXing")%>', 'value': '5'}
			],
			resultStatusOptions: [
				{'label': '<%=rb.getString("QuanBu")%>', 'value': ''},
				{'label': '<%=rb.getString("ShengXiaoShiBai")%>', 'value': '0'},
				{'label': '<%=rb.getString("YiShengXiao")%>', 'value': '1'},
			],

			policyOptions: [
				{'label': '<%=rb.getString("QuanBu")%>', 'value': ''},
				{'label': '<%=rb.getString("TiShiBingZiDongGaiZheng")%>', 'value': 'promptAndModify'},
				{'label': '<%=rb.getString("TiShiOption")%>', 'value': 'prompt'}
			],
			statusParam: '',
			policyParam: '',
			selectDevicesUrl: '',
			selectDeviceParams: {
				timeZone:timeZone,
				searchText:''
			},
			selectDeviceSearchText: '',
			detectionForm: {
				enable: 'false',
		    	time: 0,
		    	isEveryDay: 'false',
		    	processStrategy: 'prompt',
		    	retyStrategy: '1',
		    },
		    detectionRules: {
		    	time: [{ validator: validateTime}],
		    	retyStrategy: [{ validator: validateRetyStrategy}] 
		    },
		    selectionDevices: [],

			autoDetectionShow: false,
			hourArr: [
				{'label': '0:00', 'value': 0},
				{'label': '1:00', 'value': 1},
				{'label': '2:00', 'value': 2},
				{'label': '3:00', 'value': 3},
				{'label': '4:00', 'value': 4},
				{'label': '5:00', 'value': 5},
				{'label': '6:00', 'value': 6},
				{'label': '7:00', 'value': 7},
				{'label': '8:00', 'value': 8},
				{'label': '9:00', 'value': 9},
				{'label': '10:00', 'value': 10}, 
				{'label': '11:00', 'value': 11},
				{'label': '12:00', 'value': 12},
				{'label': '13:00', 'value': 13},
				{'label': '14:00', 'value': 14},
				{'label': '15:00', 'value': 15},
				{'label': '16:00', 'value': 16},
				{'label': '17:00', 'value': 17},
				{'label': '18:00', 'value': 18},
				{'label': '19:00', 'value': 19},
				{'label': '20:00', 'value': 20},
				{'label': '21:00', 'value': 21},
				{'label': '22:00', 'value': 22},
				{'label': '23:00', 'value': 23}
			],
			defaultCheckedGroup: [],
			resultUrl: '${ctx}/task/BatchConfiguration/getBatchConfigTaskResultPageList.action',
			resultParams:{
				timeZone: timeZone,
				searchText:'',
				status: '',
				processStrategy: '',
				serialNumber: ''
			},
			resultText: '',
			resultStatus: '',
			resultPolicy: '',
			bulkSelectShow: false,
			showLog:false,
			logType:'',
			logTitle:'',
			exportAll:true,
			checkAllParam:false,
			checkedParamGroup:[],
			
			activeAll:false,
			form: {
				device: ['serial_number','host_name','band','bandwidth','EARFCNDLINUSE','plmnid','tac','tx_power','CELL_IDENTITY','PHYCELLID','rootIndex','mme_status','specialSubframe','signment'],
			},
			cellAll:false,
			freqAll:false,
			deviceCol: deviceCol.filter(function(item){ return !ignores.includes(item.code);}),
			
			deviceGroupUrl:'',
			operList:[],
			showDeviceMsg:false,
			pageSize:50,
			pageList:[50,100,200],
			selectedIds:'',
            tabActiveName:'eNB',
            gsmEnable: supportGSM
		}		
	},
	computed: {
		colAll() {
			var vm = this;
			return vm.basicAll && vm.cellAll && vm.freqAll;
		},
		basicAll() {
			var vm = this;
			return vm.deviceCol.length == vm.form.device.length;
		},
		showCols() {
			var vm = this;
			return vm.form.device;
		},
		hasConfigRole() {
			return writableMap.CODE_ENB_BATCH_CONFIG == true;
		}
	},
	watch:{
		selectionData(newVal){
            if(newVal.length == 0){
                this.bulkSelectShow = false;
            }
        }
	},
	methods:{
		// 打开已选弹窗
        openBulkSelectTable(){
            var vm = this;
            
            vm.bulkSelectShow = true;
        },
        // 关闭已选弹窗
        closeBulkSelectTable(){
            var vm = this;
			
            vm.bulkSelectShow = false;
        },
        // 设备已选表格 清空事件
        clearBulkSelected(){
            var vm = this;
            
            if(vm.activeName == 'batch'){
            	vm.$refs["ctable"].clearSelection();
			}else if(vm.activeName == 'neighborFrequencyConfig'){
				vm.$refs["neighborFrequencyTable"].clearSelection();
			}else if(vm.activeName == 'neighborCellConfig'){
				vm.$refs["neighborCellTable"].clearSelection();
			}else if(vm.activeName == 'gsmNeighborCellConfig'){
				vm.$refs["gsmNeighborCellTable"].clearSelection();
			}else if(vm.activeName == 'gnbNeighborCellConfig'){
				vm.$refs["gnbNeighborCellTable"].clearSelection();
			}
			//vm.selectionData = [];
			vm.bulkSelectShow = false;
        },
        // 设备已选表格 单个删除事件
        delBulkSelected(rows){
            var vm = this,
				activeName = vm.activeName,
				tabsCodes = {
					'batch':'ctable',
					'neighborFrequencyConfig':'neighborFrequencyTable',
					'neighborCellConfig':'neighborCellTable',
					'gsmNeighborCellConfig':'gsmNeighborCellTable',	
					'gnbNeighborCellConfig':'gnbNeighborCellTable',
				},
				rowKeyCodes = {
					'batch':'serial_number',
					'neighborFrequencyConfig':'id',
					'neighborCellConfig':'id',
					'gsmNeighborCellConfig':'id',
					'gnbNeighborCellConfig':'id',
				},
				tabs = tabsCodes[activeName],
				rowKey = rowKeyCodes[activeName];

			vm.selectionData = vm.selectionData.filter((items)=>{
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
		deviceListStatusChange(val){
			var vm = this;
			
			vm.params.status = val;
		},
		selectDevicesChange(selection){
			var vm = this;
			
	    	vm.selectionDevices = selection;
		},
		
	    //自动检测
	    DetectionConfig(id,curTab,row){ 
	    	var vm = this;
	    	
	    	if(vm.batchDeteCheck == false){
				return		 					
			}
	    	vm.selectDevicesUrl = '${ctx}/task/BatchConfiguration/getBatchConfigurationTaskPageList.action';
	    	//查看详情
    		var params = {
				timeZone: timeZone
			};
	    	axios.post('${ctx}/task/BatchConfiguration/getBatchConfigAutoDetectionStrategy.action',stringify(params)).then(function(response){
   				var data = response.data;

   				setTimeout(function(){
   					if(data.enable === undefined || data.enable === '' || data.enable === null){
						vm.detectionForm.enable = 'false';
					}else{//关闭
						vm.detectionForm.enable = data.enable;
					}
   					if(data.isEveryDay == undefined || data.isEveryDay == '' || data.isEveryDay == null){
						vm.detectionForm.isEveryDay = 'false';
					}else{//关闭
						vm.detectionForm.isEveryDay = data.isEveryDay;
					}
   					if(data.processStrategy == undefined || data.processStrategy == '' || data.processStrategy == null){
						vm.detectionForm.processStrategy = 'prompt';
					}else{//关闭
						vm.detectionForm.processStrategy = data.processStrategy;
					}
   					if(data.retyStrategy == undefined || data.retyStrategy == '' || data.retyStrategy == null){
						vm.detectionForm.retyStrategy = '1';
					}else{//关闭
						vm.detectionForm.retyStrategy = data.retyStrategy;
					}
   					if(data.time === '' || data.time === null || data.time === undefined){
   						vm.detectionForm.time = 0;
   	   				}else{
   	   					vm.detectionForm.time = data.time;
   	   				}
				},500)
	    	})

	    	vm.autoDetectionShow = true;
	    	vm.showImportCard = false;
	    },
		automicSubmit(){
			var vm = this, params = {};
			
			params.enable = vm.detectionForm.enable; // 'true/false'
			params.time = vm.detectionForm.time;
			params.isEveryDay = vm.detectionForm.isEveryDay; // 'true/false'
			params.processStrategy = vm.detectionForm.processStrategy; // 'prompt/promptAndModify'
			params.retyStrategy = vm.detectionForm.retyStrategy;
			params.timeZone = timeZone;

			vm.$refs.detectionForm.validate((valid) => {
				if(valid){
					axios.post('${ctx}/task/BatchConfiguration/updateBatchConfigAutoDetectionStrategy.action',stringify(params)).then(function(response){
	    				var data = response.data;

	    				if(data["success"]){
	    					vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success',	    			 
	    					})
                            vm.$refs.ctable.refresh();
	    				}else{
	    					vm.$message.error(data["message"])
	    				}
	    				vm.autoDetectionShow = false;
	    			})
				}
			})
		},
		automicCancel(){
			var vm = this;
			
			vm.autoDetectionShow = false;
		},
		resultQuery(){
			var vm = this;

			vm.resultParams.searchText = vm.resultText;
		},
		resultExport(){
			var vm =this;

			exportByForm("${ctx}/task/BatchConfiguration/exportBatchConfigResult.action",{
				timeZone: timeZone,
				searchText: vm.resultText,
				status: vm.resultParams.status,
				processStrategy: vm.resultParams.processStrategy,
				serialNumber: vm.resultParams.serialNumber
	         });
		},
		resultStatusChange(val){
			var vm = this;

			vm.resultParams.status = val;
		},
		resultPolicyChange(val){
			var vm = this;

			vm.resultParams.processStrategy = val;
		},
		init(){
			var vm = this,
				params = {
					timeZone:timeZone,
					searchText:'',
					status: '',
					page:1,
					rows:50,
					sort:'',
					order: ''
				};

			if(writableMap.CODE_ENB_BATCH_CONFIG){
				vm.activeName = "batch"
			}
		},
		//表格数据加载成功回调
		deviceListLoadSuccess(data) {
			var vm = this;	

			if(data) {
				if(data.rows.length > 0 && vm.activeName == 'batch'){
					vm.batchDeteCheck = true;		 					
				}else {
					//无数据时
					vm.batchDeteCheck = false;
					vm.selectSN = '';
					vm.resultParams.serialNumber = '';
				}

				vm.basicTotal = data.total;				
				vm.initTableData();
			}			
		},
		neighborCellLoadSuccess(data){
			var vm = this;	

			if(data) {
				vm.neighborCellTotal = data.total;
				vm.initTableData();
			}
		},
		//GSM 邻区
		gsmNeighborCellLoadSuccess(data){
			var vm = this;	

			if(data) {
				vm.gsmNeighborCellTotal = data.total;
				vm.initTableData();
			}
		},
		//5g 邻区
		gnbNeighborCellLoadSuccess(data){
			var vm = this;	

			if(data) {
				vm.gnbNeighborCellTotal = data.total;
				vm.initTableData();
			}
		},
		neighborFreLoadSuccess(data){
			var vm = this;

			if(data) {
				vm.neighborFreTotal = data.total;
				vm.initTableData();
			}
		},
		initTableData(){
			var vm = this;

			if(vm.basicTotal== 0 && vm.neighborCellTotal == 0 && vm.neighborFreTotal == 0 && vm.paramTotal == 0 && vm.gsmNeighborCellTotal == 0 && vm.gnbNeighborCellTotal == 0){
				vm.defaultIssueBtnShow = true;
			}else{
				vm.defaultIssueBtnShow = false;
			}				
		},
		batchSelect(selection){
			var vm = this;

	    	vm.selectionData = selection;
		},
		tabClick(){
			var vm = this;
			
			vm.selectionData = [];	
				
        	if(vm.activeName == 'batch'){	    		
				vm.$refs.ctable.clearSelection();
	    	}else if(vm.activeName == 'neighborCellConfig'){
				vm.$refs.neighborCellTable.clearSelection();
	    	}else if(vm.activeName == 'neighborFrequencyConfig'){
				vm.$refs.neighborFrequencyTable.clearSelection();
	    	}else if(vm.activeName == 'gsmNeighborCellConfig'){
				vm.$refs.gsmNeighborCellTable.clearSelection();
			}else if(vm.activeName == 'gnbNeighborCellConfig'){
				vm.$refs.gnbNeighborCellTable.clearSelection();
			}		

			vm.refreshTable();
			vm.autoDetectionShow = false;
	    	vm.showImportCard = false;
		},
		
		//右上角下发按钮点击
		issueBtn(type){
			var vm = this;
			
			if(type == 'all'){
				vm.activeAll = true;
			}else{
				vm.activeAll = false;
				if(vm.selectionData.length == 0){
	                return
	            }
			}

			vm.issueRuleForm.isReboot = '0';
			vm.showIssueCard = true;
		},
		//确认下发
		issueConfirm(){
			var vm = this, curIds = '', params = {}; 

			if(vm.selectionData.length != 0){
				curIds = vm.selectionData.map(function(item){
					return item.id
				}).join(',');
			}

			if(vm.activeName == 'batch'){	  
				params.ids = curIds;
				if(vm.activeAll){
					params.executeType = 'All'
				}
				params.isReboot = vm.issueRuleForm.isReboot;
			}else if(vm.activeName == 'neighborCellConfig'){
				params.cellIds = curIds;
			}else if(vm.activeName == 'neighborFrequencyConfig'){
				params.freqIds = curIds;
	    	} else if(vm.activeName == 'gsmNeighborCellConfig'){
				params.gsmFreqIds = curIds;
			} else if(vm.activeName == 'gnbNeighborCellConfig'){
				params.gnbFreqIds = curIds;
			}

			vm.$refs.issueRuleForm.validate((valid) => {
				if(valid){					
					axios.post('${ctx}/task/BatchConfiguration/executeBatchConfiguration.action',stringify(params)).then(function(response){
			    		var data = response.data;

			    		if(data["success"]){			    			
							updateTimer = setInterval(function(){
								
								var configPageCtn = $("#configurationPlan");			
								if(!configPageCtn.length) {
									clearInterval(updateTimer);
									return;
								} 
		    		    		
		    		    		vm.issueBtnShow = true;
	    						vm.issueDisabled = true;
	    						vm.suspendBtnShow = false;
	    						vm.suspendDisabled = false;
			    				axios.post('${ctx}/task/BatchConfiguration/getBatchConfigExecuteStatus.action').then(function(response){
			    		    		var data = response.data;
		    						
			    		    		if(data == true){
			    		    			//true 真正下发成功：下发按钮 隐藏，置灰；显示终止按钮，不置灰
			    						vm.issueBtnShow = true;
			    						vm.issueDisabled = false;
			    						
			    						vm.suspendBtnShow = false;
			    						vm.suspendDisabled = false;
			    						
			    						vm.togetherRefreshTable();
			    						clearInterval(updateTimer);
			    					}else{
			    						vm.issueBtnShow = false;
			    						vm.issueDisabled = false;
			    						
			    						vm.suspendBtnShow = true;
			    						vm.suspendDisabled = false;
			    					}		    		
			    		    	})
			    			},6000)
						}else{
							//如果下发失败:下发按钮显示，不置灰；终止按钮不显示，不置灰
							vm.issueBtnShow = true;
							vm.issueDisabled = false;			    			
							vm.$message.error(data["message"])																			
						}
			    		vm.closeIssueConfirm();
						vm.commonDelClearSelectionData();
			    		vm.togetherRefreshTable();
			    	})
				}
			})
		},
		closeIssueConfirm(){
			var vm = this;

			vm.issueRuleForm.isReboot = '0';
			vm.showIssueCard = false;
		},
		//右上角-终止按钮点击
		terminateIssueBtn(){
			var vm = this;
			var h = this.$createElement;
			var msg = h('div',null,[
				h('div',null,'<%=rb.getString("QueRenZhongZhiSuoYouPeiZhi")%>'),
				h('div',null,'<%=rb.getString("DengDaiDeRenWuJiangZhongZhi")%>')
			])

			vm.$confirm(msg,'<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {	
	    		clearInterval(updateTimer);

	    		axios.post('${ctx}/task/BatchConfiguration/updateBatchSuspend.action').then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			
		    			suspendTimer = setInterval(function(){
		    				var configPageCtn = $("#configurationPlan");			
							if(!configPageCtn.length) {
								clearInterval(suspendTimer);
								return;
							}
	    		    		vm.issueBtnShow = false;
							vm.issueDisabled = true;
							
							vm.suspendBtnShow = true;
							vm.suspendDisabled = true;
							
		    				axios.post('${ctx}/task/BatchConfiguration/getBatchSuspendStatus.action').then(function(response){
		    		    		var data = response.data;

		    		    		if(data == true){
		    		    			//终止成功:下发按钮 显示,不置灰，终止按钮隐藏，不置灰
		    		    			vm.issueBtnShow = true;
		    						vm.issueDisabled = false;
		    						
		    						vm.suspendBtnShow = false;
		    						vm.suspendDisabled = false;
		    								    						
		    						vm.togetherRefreshTable();
		    						clearInterval(suspendTimer);
		    					}		    		
		    		    	})
		    			},6000)			
					}else{
						//如果终止失败，终止按钮 显示，不置灰；下发按钮,隐藏，不置灰；
						vm.suspendBtnShow = true;
		    			vm.suspendDisabled = false;
		    			
						vm.$message.error(data["message"])
					}
					vm.commonDelClearSelectionData();
					vm.togetherRefreshTable();
		    	})

	    	}).catch(() => {
	    		
	    	})
		},

		// 打开导入弹出框
		importFileBtn(){
			var vm= this;

			vm.showImportCard = true;
			vm.autoDetectionShow = false;
		},
		//导入
		/**
		* 文件上传成功函数 
		* @param res{object}   返回信息
		* @param file{object}  文件信息
		*/
		checkFile(res,file){    //发送请求，校验device文件内容 
			var vm = this;
		
			if(res.success){
				if(res.suc_count>0){
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
				}else {
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("ShiBai")%>'
					});
				}	

				vm.togetherRefreshTable();
				vm.closeFileSelect();
				vm.showImportCard = false;
			}else{
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.upload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		fileChange(file,fileList){ 
			var vm = this;

			vm.fileName = file.name;
			vm.fileParams.FileName = file.name;
		},
		// 选择文件
		fileSelect(){  
			var vm = this;

			vm.$refs.upload.clearFiles();
			vm.$refs['file_up'].click();
		},
		// 移除导入文件
		closeFileSelect(){
			var vm = this;

			vm.fileName = '';			
			vm.$refs.upload.clearFiles();
		},
		/**
		* 文件上传之前
		* @param file{object}   文件信息
		*/ 
		beforeUpload(file){
			var vm = this, importUrl, fileName = file.name,fileSize = file.size,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};

			fd.append('importType',vm.importRuleForm.importType);//导入类型
			fd.append('fileSize',fileSize);//文件大小
			fd.append('uploadFile',file); //文件流
			vm.importLoading = true;

			axios.post('${ctx}/task/BatchConfiguration/importBatchConfigurationInfos.action',fd,config).then(function(res){
				var data = res.data;

				if(data["success"]){	
					vm.$message.success('<%=rb.getString("ChengGong")%>');
					vm.togetherRefreshTable();
					vm.closeImportDevice();
					vm.init(); 
				}else{
					vm.closeImportDevice();
        			$("#failureTextBatch").text(data["msg"]);
					$("#winDowloadFailureFileBatch").window("setTitle", " <%=rb.getString("XinXi")%>");
					$("#winDowloadFailureFileBatch").window("open");
					//如果data.downloadFailFile为true,需要将$("#winDowloadFailureFileBatch")窗口中的下载按钮显示，否则隐藏
					if(data["downloadFailFile"]){
						$("#winDowloadFailureFileBatch .linkbutton").show();
					}else{
						$("#winDowloadFailureFileBatch .linkbutton").hide();
					}
				}
				vm.importLoading = false;
			})
			
			return false;
		},
		//确定导入
        uploadDevice() {
			var vm = this;

			vm.$refs.importRuleForm.validate((valid) => {
                if (valid) {
                	vm.$refs.upload.submit();                   	
                }
            }) 				
		},
		// 关闭导入弹出框
		closeImportDevice(){
			var vm = this;	

			vm.fileList = [];
			vm.fileName = '';
			vm.importRuleForm = {
				importType: "append",
				filePath: ""
			}
			vm.$refs.importRuleForm.resetFields();
			vm.showImportCard = false;
		},
		//下载模板
		exportTemplate(){
			exportByForm('${ctx}/task/BatchConfiguration/importBatchConfigurationTemp.action', {});
		},

		deviceGroupSelect(val) {
			var vm = this;

			vm.selectedIds = val;
			
			if(val.length == 0){
				vm.showDeviceMsg = true;
			}else{
				vm.showDeviceMsg = false;
			}
			
		},
		colAllChange(val) {
			var vm = this;

			vm.deviceAllChange(val);
			vm.cellAllChange(val);
			vm.freqAllChange(val);
		},
		deviceAllChange(val) {
			var vm = this,
				fields = vm.deviceCol.map(function(item){
					return item.code;
				}),
				filters = vm.deviceCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.device = val?fields:filters;
		},
		cellAllChange(val){
			var vm = this;

			vm.cellAll = val;
		},
		freqAllChange(val){
			var vm = this;

			vm.freqAll = val;
		},
		//导出
		exportFileBtn(){
			var vm = this,
				params = {
					TimeZone: timeZone,
					operator_codes: operator_code,
					content: '',
					isDual:false,
					isMonitor:true
				};

			var columns = vm.showCols,groupid=[],
				content = columns.map(function(item){ return item;}).join(',');

			params.content = content;
			
			if(vm.selectedIds.length == 0){
				vm.showDeviceMsg = true;
				return;
			}else {
				vm.selectedIds.map(function(item){
					groupid.push(item.groupId)
				})
				params.group_id = groupid.join(",");
			}
						
			exportByForm('${ctx}/task/BatchConfiguration/exportCellsToExcel.action', params);

			vm.closeExport();
		},

		closeExport(){
			document.body.click();
		},
		clearLogs(){
			var vm = this;
			var url = '${ctx}/task/BatchConfiguration/deleteBatchConfigTaskRecord.action'
			vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(url).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
						vm.$message({
							message:'<%=rb.getString("ShanChuChengGong")%>',
							type:'success',
						})
                        vm.$refs.resultCtable.refresh();
					}else{
						vm.$message.error(data["message"])
					}
		    	}) 
	    	}).catch(() => {
	    		
	    	})
		},
		//配置表格搜索
		query(){ 
			var vm = this;

			vm.params.searchText = vm.configSearchText;
		},
		selectDeviceQuery(){ 
			var vm = this;

			vm.selectDeviceParams.searchText = vm.selectDeviceSearchText;
		},
	    //邻区配置表格搜索
	    neighborCellQuery(){ 
			var vm = this;

			vm.neighborCellParams.searchText = vm.neighborCellSearchText;
		},
		//gsm邻区配置表格搜索
	    gsmNeighborCellQuery(){ 
			var vm = this;

			vm.gsmNeighborCellParams.searchText = vm.gsmNeighborCellSearchText;
		},
		//5g 邻区配置表格搜索
	    gnbNeighborCellQuery(){ 
			var vm = this;

			vm.gnbNeighborCellParams.searchText = vm.gnbNeighborCellSearchText;
		},
		//临频配置表格搜索
		neighborFrequencyQuery(){
			var vm = this;

			vm.neighborFrequencyParams.searchText = vm.neighborFrequencySearchText;
		},
		
		//点击页面其他地方菜单收起
		handerClose(){
			var vm = this;
			
        	if(vm.activeName == 'batch'){	    		
        		vm.$refs.menu.hide();
	    	}else if(vm.activeName == 'neighborCellConfig'){
	    		vm.$refs.neighborCellMenu.hide();
	    	}else if(vm.activeName == 'neighborFrequencyConfig'){
	    		vm.$refs.neighborFrequencyMenu.hide();
	    	}else if(vm.activeName == 'gsmNeighborCellConfig'){
				vm.$refs.gsmNeighborCellMenu.hide();
			}else if(vm.activeName == 'gnbNeighborCellConfig'){
				vm.$refs.gnbNeighborCellMenu.hide();
			}
	    },
	    templateRowClick(row){ 
	    	var vm = this;
	    		    	
	    	if(vm.activeName == 'batch'){	   
	    		vm.selectSN = row.serial_number;
	    		vm.resultParams.serialNumber = row.serial_number;
	    		vm.logType = '';
	    		vm.showLog = true;
	    		vm.logTitle = '<%=rb.getString("FaSongJieGuo")%>'
	    	}
	    	
	    	if (row == 'all') {
	    		vm.resultParams.serialNumber = '';
	    		vm.logType='all';
	    		vm.logTitle = '<%=rb.getString("Log")%>'
	    		vm.showLog = true;
	    	}
	    },
		/**
		 * 操作栏点击展开下拉菜单
		 * @param row:点击的数据
		*/
	    optClick(row,ev){ 
		    var vm = this, editFlag = false, delFlag = false, issueFlag = false, offlineFlag = false;

	    	vm.rowData = row;

	    	//基础配置表格：0:成功（已生效）；1：失败（生效失败）；2：进行中（生效中）；3：未生效
	    	//邻区，临频配置表格：success:成功（已生效）；fail：失败（生效失败）；running：进行中（生效中）；unexecute：未生效 
	    	//生效中，修改，下发,删除都不可点
	    	if(row.status == 'running' || row.status == '2'){
	    		editFlag = true;
	    		delFlag = true;	    		
	    		issueFlag = true;
	    	}
	    	//状态 4-离线待执行操作
	    	if(row.status == '4'){
	    		offlineFlag = false;
	    	}else {
	    		offlineFlag = true;
	    	}
	    	
			//注意基础配置 ，邻区配置，临频配置 相关的权限CODE
	    	vm.menus= [
	        	{label:'<%=rb.getString("PeiZhiXiaFa")%>',cls:"el-icon el-icon-operation-issued",code:'issue',disable:issueFlag},
	        	{label:'<%=rb.getString("QuXiaoShangXianZiDongXiaFa")%>',cls:"el-icon el-icon-operation-cancelAuto",code:'offline',disable:offlineFlag},
	            {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'edit',disable:editFlag},
	            {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del',disable:delFlag}
	    	]

	    	vm.$nextTick(function(){
	    		document.body.click();

	    		if(vm.activeName == "batch"){
	    			vm.$refs.menu.show(ev);
				}else if(vm.activeName == "neighborCellConfig"){
					vm.$refs.neighborCellMenu.show(ev);
				}else if(vm.activeName == 'neighborFrequencyConfig'){
					vm.$refs.neighborFrequencyMenu.show(ev);
				}else if(vm.activeName == 'gsmNeighborCellConfig'){
					vm.$refs.gsmNeighborCellMenu.show(ev);
				}else if(vm.activeName == 'gnbNeighborCellConfig'){
					vm.$refs.gnbNeighborCellMenu.show(ev);
				}
	    	});
	    },
		/**
		 * 点击菜单
		 * @param ev:点击具体数据项进行筛选
		*/
	    clickMenu(ev){
	    	var vm = this,
	    		codes = {
					issue : vm.issueConfig,
					offline: vm.offlineConfig,
					edit : vm.editConfig,
					del : vm.delConfig
				}

	    	if(codes[ev.code]){
	    		codes[ev.code](vm.rowData.id, vm.activeName, vm.rowData)
	    	}
	    },
	    /**
		 * 单个下发操作
		 * @param id:数据ID
		*/
	    issueConfig(id,curTab,row){ 
	    	var vm = this, issueUrl = '';

	    	if(curTab == "batch"){
				issueUrl = "${ctx}/task/BatchConfiguration/exeConfigById.action";
			}else if(curTab == "neighborCellConfig"){
				issueUrl = "${ctx}/task/BatchConfiguration/exeNeighbourCellById.action";
			}else if(curTab == 'neighborFrequencyConfig'){
				issueUrl = "${ctx}/task/BatchConfiguration/exeNeighbourFreqById.action";
			}else if(curTab == 'gsmNeighborCellConfig'){
				issueUrl = "${ctx}/task/BatchConfiguration/exeGSMNCellTaskById.action";
			}else if(curTab == 'gnbNeighborCellConfig'){
				issueUrl = "${ctx}/task/BatchConfiguration/exeGNBNCellTaskById.action";
			}

	    	vm.$confirm('<%=rb.getString("QueRenYaoGeiJiZhanXiaFaCanShu")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(issueUrl, stringify({
		    		id : id
		    	})).then(function(response){
		    		var data = response.data;

		    		if(data["success"]){
						vm.$message({
							message:'<%=rb.getString("ChengGong")%>',
							type:'success',
						})
                        vm.refreshTable();
					}else{
						vm.$message.error(data["message"])
					}
		    	})
	    	}).catch(() => {
	    		
	    	})	    	
	    },
	    //离线待执行操作
	    offlineConfig(id,curTab,row){ 
	    	var vm = this, commonExecuteType = '';
	    	
	    	if(curTab == "batch"){
	    		commonExecuteType = 'basis'
			}else if(curTab == "neighborCellConfig"){
				commonExecuteType = 'nCell'
			}else if(curTab == 'neighborFrequencyConfig'){
				commonExecuteType = 'nFreq'
			}else if(curTab == 'gsmNeighborCellConfig'){
				commonExecuteType = 'GSM-nCell'
			}else if(curTab == 'gnbNeighborCellConfig'){
				commonExecuteType = 'GNB-nFreq'
			}

	    	vm.$confirm('<%=rb.getString("LiXianDaiZhiXingTiShi")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/task/BatchConfiguration/cancelOnlineExecuteTask.action', stringify({
		    		id : id,
		    		executeType: commonExecuteType
		    	})).then(function(response){
		    		var data = response.data;
					
		    		if(data["success"]){
						vm.$message({
							message:'<%=rb.getString("ChengGong")%>',
							type:'success',
						})
                        vm.refreshTable();
					}else{
						vm.$message.error(data["message"])
					}
		    	})
	    	}).catch(() => {
	    		
	    	})
	    },
		/**
		 * 修改信息
		 * @param id:数据ID
		*/
	    editConfig(id,curTab,row){ 
	    	var vm = this;
	    	
			//gsm 5g 临区
			if(curTab == 'gsmNeighborCellConfig' || curTab == 'gnbNeighborCellConfig'){
				vm.slideUrl = "${ctx}/task/BatchConfiguration/toBatchConfigurationModify.action";
			}else{
				vm.slideUrl = "${ctx}/task/BatchConfiguration/toBatchConfigurationOper.action?id="+id
			}
			//vm.slideUrl = "${ctx}/task/BatchConfiguration/toBatchConfigurationImport.action?id="+id

	    	vm.slideTitle = '<%=rb.getString("XiuGai")%>'
	    	vm.slideFooter = true
	    	vm.slidePosition = 'right'
	    	vm.slideHeight = '100%'
	    	vm.slideWidth = '100%'
	    	vm.$refs.slide.showSlide(function(){
	    		vm.modal = true
	    		eventBus.$emit('modify-config',id,curTab,row)
	    	});
	    },
		/**
		 * 删除信息
		 * @param id:数据ID
		*/
	    delConfig(id,curTab,row){ 
			var vm = this, delUrl ,params = {}, msg;
			var h = this.$createElement;

			msg = h('div',null,[
				h('div',null,'<%=rb.getString("QueRenZhongZhiSuoYouPeiZhi")%>'),
				h('div',null,'<%=rb.getString("DengDaiDeRenWuJiangZhongZhi")%>')
			]);
			
			if(curTab == "batch"){
				params.id = id;
				delUrl = "${ctx}/task/BatchConfiguration/deleteBatchConfiguration.action";
			}else if(curTab == "neighborCellConfig"){
				params.ids = id;
				delUrl = "${ctx}/task/BatchConfiguration/delBatchNeighbourCell.action";
			}else if(curTab == 'neighborFrequencyConfig'){
				params.ids = id;
				delUrl = "${ctx}/task/BatchConfiguration/delBatchNeighbourFreq.action";
			}else if(vm.activeName == 'gsmNeighborCellConfig'){
				params.ids = id;
				delUrl = "${ctx}/task/BatchConfiguration/delGSMNcell.action";
			}else if(vm.activeName == 'gnbNeighborCellConfig'){
				params.ids = id;
				delUrl = "${ctx}/task/BatchConfiguration/delGNBNcell.action";
			}

	    	vm.$confirm(msg,'<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(delUrl, stringify(params)).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
						vm.$message({
							message:'<%=rb.getString("ShanChuChengGong")%>',
							type:'success',
						})
                        vm.refreshTable();
					}else{
						vm.$message.error(data["message"])
					}
		    		vm.commonDelClearSelectionData();
		    	}) 
	    	}).catch(() => {
	    		
	    	})
	    },
	    // 批量删除
	    deleteCells(type){
	    	var vm = this, delUrl = '', curDelId = '', params = {}, msg='';	
	    	var h = this.$createElement;

			msg = h('div',null,[
				h('div',null,'<%=rb.getString("QueRenYiChuSuoXuanPeiZhi")%>'),
				h('div',null,'<%=rb.getString("ShanChuZhiXingPeiZhi")%>')
			]);
			
			if(type == 'all'){
				params.executeType = 'All';
				msg = h('div',null,[
					h('div',null,'<%=rb.getString("QueRenYiChuSuoYouPeiZhi")%>'),
					h('div',null,'<%=rb.getString("ShanChuZhiXingPeiZhi")%>')
				]);
			}else if(vm.selectionData.length != 0){
	    		curDelId = vm.selectionData.map(function(item){
	    			return item.id
				}).join(',')
			}
			
			if(vm.activeName == "batch"){
				params.id = curDelId;
				delUrl = "${ctx}/task/BatchConfiguration/deleteBatchConfiguration.action";
			}else if(vm.activeName == "neighborCellConfig"){
				params.ids = curDelId;
				delUrl = "${ctx}/task/BatchConfiguration/delBatchNeighbourCell.action";
			}else if(vm.activeName == 'neighborFrequencyConfig'){
				params.ids = curDelId;
				delUrl = "${ctx}/task/BatchConfiguration/delBatchNeighbourFreq.action";
			}else if(vm.activeName == 'gsmNeighborCellConfig'){
				params.ids = curDelId;
				delUrl = "${ctx}/task/BatchConfiguration/delGSMNcell.action";
			}else if(vm.activeName == 'gnbNeighborCellConfig'){
				params.ids = curDelId;
				delUrl = "${ctx}/task/BatchConfiguration/delGNBNcell.action";
			}

	    	vm.$confirm(msg,'<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(delUrl, stringify(params)).then(function(response){
		    		var data = response.data;

		    		if(data["success"]){
						vm.$message({
							message:'<%=rb.getString("ShanChuChengGong")%>',
							type:'success',
						})
                        vm.refreshTable();
					}else{
						vm.$message.error(data["message"])
					}
		    		vm.commonDelClearSelectionData();
		    	}) 
	    	}).catch(() => {
	    		
	    	})
	    },
	    //点击删除，批量删除后，将数据清空，底部弹窗关闭
	    commonDelClearSelectionData(){	    	
			var vm = this;
			
			vm.selectionData = [];	

        	if(vm.activeName == 'batch'){	    		
				vm.$refs.ctable.clearSelection();
	    	}else if(vm.activeName == 'neighborCellConfig'){
				vm.$refs.neighborCellTable.clearSelection();
	    	}else if(vm.activeName == 'neighborFrequencyConfig'){
				vm.$refs.neighborFrequencyTable.clearSelection();
	    	}else if(vm.activeName == 'gsmNeighborCellConfig'){
				vm.$refs.gsmNeighborCellTable.clearSelection();
			}else if(vm.activeName == 'gnbNeighborCellConfig'){
				vm.$refs.gnbNeighborCellTable.clearSelection();
			}
	    },
	    
		// 关闭修改页面
	    closeSlide(){ 
			var vm = this;

	    	vm.$refs.slide.hide();
	    	vm.refreshTable();
	    },
	    //刷新表格数据
	    refreshTable(){
			var vm = this;

			if(vm.activeName == "batch"){
				vm.$refs.ctable.refresh();
			}else if(vm.activeName == "neighborCellConfig"){
				vm.$refs.neighborCellTable.refresh();
			}else if(vm.activeName == 'neighborFrequencyConfig'){
				vm.$refs.neighborFrequencyTable.refresh();
			}else if(vm.activeName == 'gsmNeighborCellConfig'){
				vm.$refs.gsmNeighborCellTable.refresh();
			}else if(vm.activeName == 'gnbNeighborCellConfig'){
				vm.$refs.gnbNeighborCellTable.refresh();
			}
		},
		//下发 ，终止，导入统一刷新表格
		togetherRefreshTable(){
			var vm = this;	

			vm.$refs.ctable.refresh();			
			vm.$refs.neighborCellTable.refresh();			
			vm.$refs.neighborFrequencyTable.refresh();
			vm.$refs.gsmNeighborCellTable.refresh();
			vm.$refs.gnbNeighborCellTable.refresh();
		},
	    cancelSlide(){
	    	eventBus.$emit('cancel-config')
	    },
	 	// 关闭修改弹窗
	    cancelModify(){ 
			var vm = this;

	    	vm.$refs.slide.hide();
	    },
	 	// 新建保存详情
	    saveConfig(){
	    	eventBus.$emit('save-config');
	    },
	    	   
		/**
		 * 带宽 转换
		 * @param cellValue:传入的带宽数据 进行转换
		*/
	    bandWidthFmt(row,column,cellValue,index){
	    	var code = {
	    			n25 : '5',
	    			n50 : '10',
	    			n75 : '15',
	    			n100 : '20'
	    	}
	    	return code[cellValue]
	    },
		/**
		 * 字真配比 转换
		 * @param cellValue:传入的 字真配比 数据 进行转换
		*/
	    assignFmt(row,column,cellValue,index){
	    	var code = {
	    			1 : '1(DL:UL = 2:2)',
	    			2 : '2(DL:UL = 3:1)'
	    	}
	    	return code[cellValue]
	    },
		/**
		 * 频带 转换
		 * @param cellValue:传入的 频带 数据 进行转换
		*/
		earfcnFmt(row, value, index) {
			var vm = this, resList = [];
			 
            if(value) {
                value.split(',').map(function(item){
                    resList.push(vm.earfcnFormatter(item, row, index));
                });
                return resList.join(',');
            }else {
                return '';
            }
        },
        /**
         * 发射功率
         * @param cellValue:传入的 功率 数据 进行转换
        */
        txPowerFmt(row, value, index) {
            var vm = this, commonTxPower = [], txPowerLabel = '',
               	powerList = ['-20dBm','-10dBm','0dBm','1dBm','2dBm','3dBm','4dBm','5dBm','6dBm','7dBm','8dBm','9dBm','10dBm','11dBm','12dBm','13dBm','14dBm','15dBm','16dBm','17dBm','18dBm','19dBm','20dBm','21dBm','22dBm','23dBm','24dBm','25dBm','26dBm','27dBm','28dBm','29dBm','30dBm','31dBm','32dBm','33dBm','34dBm','35dBm','36dBm','37dBm','38dBm','39dBm','40dBm','41dBm','42dBm','43dBm','44dBm','45dBm','46dBm'],
                powerValList = [-20,-10,0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46];

            if(value) {
                powerValList.map((item,index)=>{
                    if(item == value){
                        txPowerLabel = powerList[index];
                    }
                })
                return txPowerLabel;
            }else {
                return '';
            }
        },
       	earfcnFormatter(value, rowData, rowIndex){
            if(value == null){
                return "";
            }
            var EARFCN = value;//正常显示的频点值  还需要将此值转换成频率
            var frequency = 0;
            if (EARFCN >= 36000 && EARFCN <= 36199) { //tdd-band 33
                frequency = 1900 + 0.1 * (EARFCN - 36000);
            } else if (EARFCN >= 36200 && EARFCN <= 36349) { //tdd-band 34
                frequency = 2010 + 0.1 * (EARFCN - 36200);
            } else if (EARFCN >= 36350 && EARFCN <= 36949) { //tdd-band 35
                frequency = 1850 + 0.1 * (EARFCN - 36350);
            } else if (EARFCN >= 36950 && EARFCN <= 37549) { //tdd-band 36
                frequency = 1930 + 0.1 * (EARFCN - 36950);
            } else if (EARFCN >= 37550 && EARFCN <= 37749) { //tdd-band 37
                frequency = 1910 + 0.1 * (EARFCN - 37550);
            } else if (EARFCN >= 37750 && EARFCN <= 38249) { //tdd-band 38
                frequency = 2570 + 0.1 * (EARFCN - 37750);
            } else if (EARFCN >= 38250 && EARFCN <= 38649) { //tdd-band 39
                frequency = 1880 + 0.1 * (EARFCN - 38250);
            } else if (EARFCN >= 38650 && EARFCN <= 39649) { //tdd-band 40
                frequency = 2300 + 0.1 * (EARFCN - 38650);
            } else if (EARFCN >= 39650 && EARFCN <= 41589) { //tdd-band 41
                frequency = 2496 + 0.1 * (EARFCN - 39650);
            } else if (EARFCN >= 41590 && EARFCN <= 43589) { //tdd-band 42
                frequency = 3400 + 0.1 * (EARFCN - 41590);
            } else if (EARFCN >= 43590 && EARFCN <= 45589) { //tdd-band 43
                frequency = 3600 + 0.1 * (EARFCN - 43590);
            } else if (EARFCN >= 18000 && EARFCN <= 18599) { //fdd-band 1
                frequency = 1920 + 0.1 * (EARFCN - 18000);
            } else if (EARFCN >= 0 && EARFCN <= 599) {
                frequency = 2110 + 0.1 * (EARFCN - 0);
            } else if (EARFCN >= 18600 && EARFCN <= 19199) { //fdd-band 2
                frequency = 1850 + 0.1 * (EARFCN - 18600);
            } else if (EARFCN >= 600 && EARFCN <= 1199) {
                frequency = 1930 + 0.1 * (EARFCN - 600);
            } else if (EARFCN >= 19200 && EARFCN <= 19949) { //fdd-band 3
                frequency = 1710 + 0.1 * (EARFCN - 19200);
            } else if (EARFCN >= 1200 && EARFCN <= 1949) {
                frequency = 1805 + 0.1 * (EARFCN - 1200);
            } else if (EARFCN >= 19950 && EARFCN <= 20399) { //fdd-band 4
                frequency = 1710 + 0.1 * (EARFCN - 19950);
            } else if (EARFCN >= 1950 && EARFCN <= 2399) {
                frequency = 2110 + 0.1 * (EARFCN - 1950);
            } else if (EARFCN >= 20400 && EARFCN <= 20649) { //fdd-band 5
                frequency = 824 + 0.1 * (EARFCN - 20400);
            } else if (EARFCN >= 2400 && EARFCN <= 2649) {
                frequency = 869 + 0.1 * (EARFCN - 2400);
            } else if (EARFCN >= 20650 && EARFCN <= 20749) { //fdd-band 6
                frequency = 830 + 0.1 * (EARFCN - 20650);
            } else if (EARFCN >= 2650 && EARFCN <= 2749) {
                frequency = 875 + 0.1 * (EARFCN - 2650);
            } else if (EARFCN >= 20750 && EARFCN <= 21449) { //fdd-band 7
                frequency = 2500 + 0.1 * (EARFCN - 20750);
            } else if (EARFCN >= 2750 && EARFCN <= 3449) { 
                frequency = 2620 + 0.1 * (EARFCN - 2750);
            } else if (EARFCN >= 3450 && EARFCN <= 3799) { //fdd-band 8
                frequency = 925 + 0.1 * (EARFCN - 3450);
            } else if (EARFCN >= 5010 && EARFCN <= 5179) { //fdd-band 12
                frequency = 729 + 0.1 * (EARFCN - 5010);
            } else if (EARFCN >= 5180 && EARFCN <= 5279) { //fdd-band 13
                frequency = 746 + 0.1 * (EARFCN - 5180);
            } else if (EARFCN >= 5730 && EARFCN <= 5849) { //fdd-band 17
                frequency = 734 + 0.1 * (EARFCN - 5730);
            } else if (EARFCN >= 6150 && EARFCN <= 6449) { //fdd-band 20     791-821 6150-6449  
                frequency = 791 + 0.1 * (EARFCN - 6150);
            }else if (EARFCN >= 9210 && EARFCN <= 9659) { //fdd-band 28       758-803 9210-9659
                frequency = 758 + 0.1 * (EARFCN - 9210);
            }else if(EARFCN >= 55240 && EARFCN <= 56740){
                frequency = 3550 + 0.1*(EARFCN - 55240);
            }else if(EARFCN >= 46790 && EARFCN <= 54539){
                frequency = 5150 + 0.1*(EARFCN - 46790);
            }else if(EARFCN >= 63000 && EARFCN <= 63999){
                frequency = 5150 + 0.1*(EARFCN - 63000);
            }else if(EARFCN >= 64000 && EARFCN <= 64999){
                frequency = 5725 + 0.1*(EARFCN - 64000);
            }else if(EARFCN >= 46790 && EARFCN <= 54539){
                frequency = 5150 + 0.1*(EARFCN - 46790);
            }else if(EARFCN >= 63000 && EARFCN <= 63999){
                frequency = 5150 + 0.1*(EARFCN - 63000);
            }else if(EARFCN >= 64000 && EARFCN <= 64999){
                frequency = 5725 + 0.1*(EARFCN - 64000);
            } else {
         		//throw new Exception("Please Input the right EARFCN!");
            	frequency = "--";
            }
            var showStr = EARFCN.toString()+"("+frequency.toString()+"MHz"+")";
            
            return showStr;
        },
	    frequencyFmt(row,column,cellValue,index){

	    	if(cellValue == ""){
	    		return ""
	    	}else{
	    		return translateToFre(cellValue)
	    	}
	    },
        // 初始化请求加载GSM配置
        initGsm(){
            var vm= this;

            if(vm.gsmEnable){
                $("#gsmConfiguration").load('${ctx}/task/BatchConfigurationFile/toGSMBatchConfigurationTask.action',function(data){
                    $.parser.parse(this);
                });
            }
        },
	},

	mounted(){
		this.init();
        this.initGsm();
		eventBus.$off('cancel-slide').$on('cancel-slide',this.closeSlide);
		eventBus.$off('cancel-modify').$on('cancel-modify',this.cancelModify);
	}
})
</script>