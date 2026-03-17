<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
	.switch-cover {
		height: 100%;
		width: 100%;
		position: absolute;
		top:0px;
		left: 0px;
		right: 0px;
		bottom: 0px;
		margin: auto;
		z-index: 66;
		cursor: pointer;
		opacity: 0;
	}
	#egwRegister .devicesMainBoxCls{
		display:flex;
		overflow: hidden;
		height: 100%;
		width: 100%;
		position: relative;
		background: #FFFFFF;
        border:none;
        min-width: 900px;
	}
	#egwRegister .groupMgmt {
		flex: 0 1 300px;
		height: 100%;
		border:1px solid #E9E9E9;
		border-right:none;
		position:relative;
		display:flex;
		flex-direction:column;
		box-sizing: border-box;
	}
	#egwRegister .tablesListBoxCls {
		flex: 1;
		display: flex;
		overflow: auto;
		background-color: #F7F7F7;
		box-sizing: border-box;
	}
	#egwRegister .tablesListBoxCls .tablesListMainBoxCls{
		position: relative;
		flex: 1;
		height: 100%;
		overflow: auto;
		border:1px solid #E9E9E9;
		border-radius: 0px 10px 10px 0px;
		background-color: #FFFFFF;
		box-sizing: border-box;
	}
	#egwRegister .tablesListBoxCls .tablesListRightBoxCls{
		flex: 0 1 360px;
		margin-left: 10px;
		position: relative;
		background-color: #FFFFFF;
		box-shadow: 0px 0px 10px 1px #E9EDF9;
		border-radius: 10px;
		border: 1px solid #E9EDF9;
		overflow: hidden;
		height: 100%;
		box-sizing: border-box;
	}
	#egwRegister .tablesListBoxCls .addGroupCls{
		flex: 0 1 500px !important;
	}
	#egwRegister .groupMgmt .splitTitle{
		font-size:14px;
		font-weight:bold;
		color:#363B4E;
		display:inline-block;
		height:34px;
		line-height:34px;
		padding:10px
	}
	.el-icon-circle-info:before{
		color:#CFCFCF;
	}
	.el-icon-common-download:before{
		color:#363B4E;
	}
	#egwRegister .el-upload__tip{
		color:red;
	}
	#egwRegister .curpo{
		cursor: pointer;
	}
	.el-table .cell.el-tooltip{
		min-width: 0px !important
	}
	#egwRegister .treeItemBoxCls{
		width: 100%;
		position: relative;
	}
	#egwRegister .treeItemBoxCls .operCls{
		position:absolute;
		z-index: 66;
		display: none;
	}
	.subGroupDeviceTitleCls{
		font-size: 12px;
		color: #333333;
		margin: 10px 0px;
	}
	.subGroupDeviceBoxCls .el-pairgrid-title{
		top:10px!important;
		right: 15px!important;
	}
	#egwRegister  .groupMgmt .el-input.el-input--small{
		width: 200px;
	}
	#egwRegister  .groupMgmt .el-tree-node__content{
		height: 30px;
		border-radius: 5px;
	}
	#egwRegister .groupTreeBox{
		height: calc(100% - 44px);
		overflow: auto;
		padding: 0px 5px;
	}
	#egwRegister .ItemLabelCls{
		display: inline-block;
		width: 220px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	#egwRegister .tableHeadBoxCls{
		position: relative;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-right: 20px;
	}
	#egwRegister .greyIcon::before{
		color: #7A7992;
		font-size: 14px;
	}
	#egwRegister .rightItemMainBox{
		padding: 20px;
		overflow: auto;
		height: calc(100% - 140px);
	}
	#egwRegister .rightItemMainBox .el-form-item{
		margin-bottom: 20px;
	}
	#egwRegister .rightItemMainBox .el-select{
		width: 100%;
	}
	#egwRegister .rightItemMainBox .el-input{
		width: 100%;
	}
	#egwRegister .tablesListRightBoxCls .footer{
		width:100%;
		border-top:1px solid #E9E9E9;
		position:absolute;
		bottom:1px;
		height:50px;
		background:#FFFFFF;
		z-index:99;
		display: flex;
		align-items: center;
		border-radius: 0px 0px 10px 10px;
	}
	#egwRegister .rightOutBoxHeadCls{
		height: 50px;
		display: flex;
		align-items: center;
		font-weight: 600;
		font-size: 14px;
		justify-content: space-between;
		padding: 0px 20px;
		border-bottom: 1px solid #E9EDF9;
	}
	#egwRegister .deviceListTableCls{
		width: 100%;
		height: 100%;
		overflow: auto;
	}
	#egwRegister .tipText{
		display:flex;           
		color: rgba(0, 0, 0, 0.32);		
		font-size:12px;
	}
	#egwRegister .tipText .infoTip { margin-top: 2px; }
	#egwRegister .tipText .infoTip:before{
		color: rgba(0, 0, 0, 0.32);		
		font-size:14px;
		margin-right:6px;
	}
	#egwRegister .grayIcon::before{
		color: #7A7992;
		font-size: 18px;
	}
	#egwRegister .importFileItem .el-input-group__append{
		padding: 0px 10px;
	}
	#egwRegister .importFileItem .el-input__suffix{
		top:4px;
	}
	#addDeviceForm .el-radio--small.is-bordered{
		padding: 8px 10px 0 8px;
		border-radius: 3px;
		height: 30px;
	}
	#addGroupForm .el-checkbox.is-bordered.el-checkbox--small{
		padding: 7px 10px 0 8px;
		border-radius: 3px;
		height: 30px;
	}
	.deviceExportPopoverMainBoxCls{
		padding: 0px 20px 20px 20px;
	}
	.deviceExportPopoverMainBoxCls .el-checkbox+.el-checkbox{
		margin-left: 20px;
	}
	#addGroupForm .addGroupDeviceBoxCls{
		height: 378px;
		width: 100%;
		box-sizing: border-box;
		overflow: hidden;
	}
	#addGroupForm .tableCalss{
		height: 520px;
	}
	#addGroupForm .addGroupDeviceBoxCls .newTabs .el-tabs__header{
		border:1px solid #D5DCEC;
		border-radius: 8px 8px 0px 0px;
		border-bottom: 0;
	}
	#addGroupForm .addGroupDeviceBoxCls .el-ctable-toolbar .el-select{
		width: 200px;
	}
	#addGroupForm .addGroupDeviceBoxCls .newTabs .el-tabs__header .el-tabs__item{
		font-size: 12px;
	}
	#addGroupForm .el-tabs__new-tab{
		display: none;
	}
	#addGroupForm .el-tabs__item{
		box-shadow: none!important;
	}
	#addGroupForm .el-tabs__content{
		padding-top:37px;
		top: -37px; 
	}
	#addGroupForm .el-pairgrid-title{
		top: -30px;
		right: 10px!important;
	}
	#addGroupForm  .pairgrid-right{
		max-width: 400px!important;
		min-width: 80%!important;
		top: 0px!important;
		overflow: hidden!important;
	}
	#addGroupForm .rightQueryBoxCls{
		max-width: 250px!important;
		margin-left: 20px;
	}
	#addGroupForm .rightQueryBoxCls .pairgrid-query{
		max-width: 200px;
	}
	#addGroupForm .el-pagination .el-select .el-input{
		margin: 0px;
	}
	#addGroupForm .queryGroup{
		height: 26px;
		margin-left: 0px;
		padding: unset;
		padding-right: 10px;
		box-sizing: border-box;
	}
	#addGroupForm .queryGroup input{
		width: 190px;
		height: 24px;
	}
	#addGroupForm .el-pagination  .el-pagination__jump,#addGroupForm .pairgrid-left .bottom-padding{
		display: none;
	}
	#egwRegister .groupTreeBox .operCls .el-icon::before{
		color: #7A7992;
	}
	#egwRegister .groupTreeBox .el-tree-node__content:hover .operCls{
		display: block;
	}
	#egwRegister .groupTreeBox .oneGroupEditCls{
		position:absolute;
		right:5px;
		top:5px;
	}
	#egwRegister .groupTreeBox .oneGroupEditCls .el-icon::before{
		font-size: 14px;
		color: #000000;
	}
	.importResultWarp .importResultDivs {
	    margin-bottom: 16px;
	}
	.importResultWarp .downloadBtnTips {
	    display: block;
	    margin: 40px auto 0;
	}
	.el-tooltip__popper.greenBg {
		background: #b2e8bd;
		margin: 0;
	}
	 .el-tooltip__popper.redBg {
		background: #f1cfcf;
		margin: 0;
	}
	#egwRegister .rightBoxBtnCls{
		position: absolute;
		right: 15px;
		top: 5px;
		display: flex;
	}
	#egwRegister .rightBoxBtnCls > div{
		position: relative;
		margin-right: 10px;
	}
	.with-options ul {
		padding-top: 40px;
	}
	.s-opts {
		display: block !important;
		width: 100%;
		position: absolute;
		top: 0px;
		padding: 5px 0px !important;
		z-index: 10;
		background-color: #fff;
	}
	.s-opts.hover {
		background-color: #fff !important;
	}
	.s-opts > div {
		padding: 0 20px;
	}
	.s-opts .option-bt {
		padding: 0 5px;
		font-weight: normal;
		border-radius: 3px;
		background-color: rgba(255, 70, 20, 0.1);
	}
    #egwRegister .ruleListTableCls{
        width: 100%;
        height: 100%;
        overflow: auto;
    }
    #egwRegister .switchBoxCls-ctn {
        display: inline-block;
        position: relative;
        cursor: pointer;
    }
    #egwRegister .switchBoxCls-ctn::before {
        content: '';
        position: absolute;
        top: 0;
        right: 0;
        bottom: 0;
        left: 0;
        z-index: 100;
    }
    #egwRegister .nameContainsContentBoxCls{
        display: flex;
        align-items: center;
        margin-bottom: 10px;
    }
    #egwRegister .nameContainsContentBoxCls:last-child{
        margin-bottom: 0px;
    }
    #egwRegister .nameContainsContentBoxCls .el-select{
        width: 120px;
    }
    #egwRegister .nameContainsContentBoxCls .el-input-group__prepend div.el-select .el-input__inner{
        border: 0;
    }
    #egwRegister .nowInputNameContainsContentBoxCls{
        width: 100%;
        font-size: 12px;
        color: rgba(0, 0, 0, 0.32);
        margin-bottom: 20px;
        word-wrap: break-word;
        word-break: break-all;
        white-space: normal;
        line-height: 1.5;
    }
    /* 输入框错误状态样式 */
    #egwRegister .nameContainsContentBoxCls .input-error .el-input__inner{
        border-color: #F56C6C !important;
    }
    #egwRegister .nameContainsContentBoxCls .input-error.is-focus .el-input__inner{
        border-color: #F56C6C !important;
    }
    #egwRegister .addNameContainsContentBoxCls{
        height: 30px;
        width: 100%;
        font-size: 12px;
        color: var(--main-color);
        display: flex;
        align-items: center;
        justify-content: center;
        border: 1px solid var(--main-color);
        border-radius: 5px;
    }
    #egwRegister .addNameContainsContentBoxCls:hover{
        background-color:rgba(var(--main-color-rgba1),0.08);
        cursor: pointer;
    }
    #egwRegister .nameContainsContentBoxCls .el-icon-circle-close:before{
        content: "\e6fb";
    }
    /* remark列label自定义样式 */
    #egwRegister .remarkHeaderCls,#egwRegister .remarkHeaderCls div {
        padding-left: unset;
        line-height: unset;
    }
    #egwRegister .remarkHeaderCls .el-input__suffix {
        line-height: 30px;
    }
    #egwRegister .remarkHeaderCls .el-input--suffix .el-input__inner {
        padding-right: 50px;
        box-sizing: border-box;
    }
    #egwRegister .remarkHeaderCls .el-icon::before {
        font-size: 16px;
        color: #7A7992;
    }
    #egwRegister .deviceListTableCls .el-ctable-toolbar{
        padding: 0px !important;
    }
    #egwRegister .deviceListTableCls .toolbarHeadBtnBoxCls{
        padding-right: 160px;
    }
</style>
<div class="overflow-cls" id="egwRegister">
    <div class="devicesMainBoxCls">
        <!--左侧设备组-->
        <div class="groupMgmt">
            <!-- 按钮  添加设备组 -->
            <div class="newIconBoxCls-bt" v-if="optDeviceShow" style="right:20px;top:12px;" @click="addDeviceGroup" tip="<%=rb.getString("TianJia")%>">
                <span class="el-icon el-icon-plus"></span>
            </div>
            <div class="splitTitle"><%=rb.getString("SheBeiZu")%></div>
            <div style="height: calc(100% - 54px);">
                <el-query type="normal" @query="queryGroupList" placeholder="<%=rb.getString("SheBeiZuMingCheng")%>" style="margin-bottom:10px;"></el-query>
                <div class="groupTreeBox">
                    <el-tree 
                        ref="groupTree"
                        :data="groupData"
                        node-key="id"
                        :default-expanded-keys="defaultexpandedKeys"
                        :default-checked-keys="defaultCheckedKeys"
                        :props= "{label:'group_name'}"
                        :highlight-current="true"
                        @node-click="groupRowClick"
                    >
                        <div class="treeItemBoxCls" slot-scope="{ node,data }">
                            <div v-if="data.children">
                                <span v-show="data.isEdit == 'false'" class="ItemLabelCls" :title="node.label">{{node.label}}</span>
                                <el-input v-show="data.isEdit == 'true'" class="ItemLabelCls" v-model="data.group_name"></el-input>
                                <div class="operCls" style="right:5px;top:0px;" v-show="data.isEdit == 'false' && optDeviceShow">
                                    <span class="el-icon el-icon-operation-more-circle " @click="groupOpClick(node,data,event)" v-clickoutside="handerClose"></span>
                                    <span class="el-icon el-icon-plus" @click="addSubGroup(node,data,event)" v-clickoutside="handerClose"></span>
                                </div>
                                <div class="oneGroupEditCls" v-show="data.isEdit == 'true'">
                                    <span class="el-icon el-icon-operation-defaultBeta" @click="oneGroupEditSubmit(node,data,event)" v-clickoutside="handerClose"></span>
                                    <span v-show="data.children" class="el-icon el-icon-deactivate" @click="oneGroupEditClose(node,data,event)" v-clickoutside="handerClose"></span>
                                </div>
                            </div>
                            <div v-else>
                                <span class="ItemLabelCls" :title="node.label">{{node.label}}</span>
                                <div class="operCls" style="right:5px;top:0px;">
                                    <span class="el-icon el-icon-operation-more-circle " @click="groupOpClick(node,data,event)" v-clickoutside="handerClose"></span>
                                </div>
                            </div>
                        </div>
                    </el-tree>
                </div>
                <el-cmenu ref="menuGroup" :data="menusGroup" @click="clickMenu"></el-cmenu>
            </div>
        </div>
        <!--右侧侧设备列表-->
        <div class="tablesListBoxCls">
            <div class="tablesListMainBoxCls">
                <div class="rightBoxBtnCls">
                    <!-- 按钮   添加设备 -->
                    <div class="newIconBoxCls-bt" v-if="optDeviceShow" @click="addDevice" tip="<%=rb.getString("TianJia")%>">
                        <span class="el-icon el-icon-plus"></span>
                    </div>
                    <!-- 按钮   配置规则 -->
                    <div class="newIconBoxCls-bt" v-if="optDeviceShow && deviceType != 'WCG'" @click="configuraRule" tip="<%=rb.getString("PeiZhiGuiZe")%>">
                        <span class="el-icon el-icon-batchInput"></span>
                    </div>
                    <!-- 按钮   导出 -->
                    <div class="newIconBoxCls-bt" @click="exportDevice" tip="<%=rb.getString("DaoChu")%>">		
                        <span class="el-icon-operation-export el-icon"></span>
                    </div>
                    <!-- 按钮   回收站 -->
                    <div class="newIconBoxCls-bt" @click="openDeviceRecycleBin" tip='<%=rb.getString("HuiShouZhan")%>' v-if="deviceType != 'WCG'">		
                        <span class="el-a-icon-Recyclebin el-icon"></span>
                    </div>
                </div>
                <!--<el-popover ref="deviceExportPopover" trigger="click" title="<%=rb.getString("DaoChu")%>" placement="bottom-start"  @hide="deviceExportPopoverHide">
                    <div class="deviceExportPopoverMainBoxCls">
                        <div style="color:#9E9E9E;margin-bottom:10px;">Select the NE to export in device list.</div>
                        <div style="height:20px;">
                            <el-checkbox-group v-model="exportDeviceType" @change="exportDeviceTypeChange">
                                <el-checkbox v-for="(item,index) in deviceTypeList" :label="item.label" :key="index"></el-checkbox>
                            </el-checkbox-group>
                            <div v-show='deviceExportErrorShow' style="color:#FA5555;font-size:12px;">Please select the NE to export</div>
                        </div>
                        <div class="buttonGroup">
                            <el-button size="mini" type="primary" @click="exportDevice"><%=rb.getString("QueDing")%></el-button>
                            <el-button size="mini" @click="deviceExportPopoverHide"><%=rb.getString("QuXiao")%></el-button>
                        </div>	
                    </div>
                    <div slot="reference" class="newIconBoxCls-bt" style="right:20px;top:5px;" tip="<%=rb.getString("DaoChu")%>">		
                        <span class="el-icon-operation-export el-icon"></span>
                    </div>
                </el-popover>-->
                <div class="deviceListTableCls">
                    <el-ctable 
                        id="deviceTable"
                        ref="ctableDevice" 
                        :url="deviceUrl" 
                        :height="height" 
                        :row-key="deviceTableRowKey" 
                        :query-params="params_device"
                        pagination="true" 
                        :rownumber=true 
                        :limit="limitBatch"
                        @selection-change='batchSelect'
                        >
                        <template slot="toolbar" style="padding: 0px;">
                            <div style="position:relative;">
                                <div class="toolbarHeadBtnBoxCls">
                                    <div v-show="optDeviceShow" class="selectBlukBoxCls">
                                        <div class="selectMain">
                                            <div style="font-weight:550;margin-left:20px;white-space: nowrap;">{{rowDataGroup.group_name}}</div>
                                            <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                                <span class="el-icon-selected el-icon"></span>
                                                <span class="bulkSelectNumBoxCls">( {{deviceSelection.length}} )</span>
                                            </div>
                                            <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                                                <div class="selectBoxTitle">
                                                    <span><%=rb.getString("YiXuan")%></span>
                                                    <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                                </div>
                                                <div class="selectBoxMain">
                                                    <div class="tableInfoCls">
                                                        <div class="tableInfoHeader">
                                                            <div>{{selectedDialogTitle}}</div>
                                                            <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
                                                        </div>
                                                        <el-ctable 
                                                            id="bulkSelectTable" 
                                                            ref="bulkSelectTable" 
                                                            :data="deviceSelection" 
                                                            :showHeader="false"
                                                            :rownumber="false"
                                                            :front-pagination="true"
                                                            :row-key="deviceTableRowKey"
                                                            height="270px" pagination="true" >
                                                            <el-table-column prop="alarm_id" v-if="false"></el-table-column>
                                                            <el-table-column width="588">
                                                                <template slot-scope="scope" >
                                                                    <div class="tableItemCls">
                                                                        <span v-if="deviceType == 'eNB'">{{scope.row.serial_number}}</span>
                                                                        <span v-if="deviceType == 'gNB'">{{scope.row.serial_number}}</span>
                                                                        <span v-if="deviceType == 'CPE'">{{scope.row.macaddress}}({{scope.row.serial_number}})</span>
                                                                        <span v-if="deviceType == 'WCG'">{{scope.row.egwSn}}</span>
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
                                    <div v-if="optDeviceShow" :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="movecells">
                                        <span class="el-icon el-icon-moveGroup"></span>
                                        <span><%=rb.getString("YiDongDaoSheBeiZu")%></span>
                                    </div>
                                    <div v-if="optDeviceShow && detectable" :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchSetDetectEnable">
                                        <span class="el-icon el-icon-status-enable"></span>
                                        <span><%=rb.getString("JianCeQiYong")%></span>
                                    </div>
                                    <div v-if="optDeviceShow && detectable" :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchSetDetectDisable">
                                        <span class="el-icon el-icon-status-disable"></span>
                                        <span><%=rb.getString("JianCeJinYong")%></span>
                                    </div>
                                    <div v-if="optDeviceShow && deviceType != 'WCG'" :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="recycleCells">
                                        <span class="el-icon el-a-icon-Recyclebin"></span>
                                        <span><%=rb.getString("HuiShouZhan")%></span>
                                    </div>
                                    <div v-if="optDeviceShow" :class="deviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="deleteCells">
                                        <span class="el-icon el-icon-operation-delete"></span>
                                        <span><%=rb.getString("PiLiangShanChu")%></span>
                                    </div>
                                </div>
                                <div class="tableHeadBoxCls" style="position:relative;">
                                    <div id="tableHeadQuery" class="tableHeadQueryBoxCls">
                                        <div class="headQueryBox">
                                            <div class="queryGroup">
                                                <el-input v-model="search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
                                                <i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
                                            </div>
                                        </div>
                                    </div>
                                    <!--<el-radio-group size="mini" v-model='deviceType' class="commonRadioButton" @change="deviceTypeChange">
                                        <el-radio-button v-for="item in deviceTypeList" :label="item.label">{{item.label}}</el-radio-button>
                                    </el-radio-group>-->
                                </div>
                            </div>
                        </template>
                        <!--设备列表-->
                        <el-table-column width="50" type="selection" prop="ck" v-if="optDeviceShow && groupWritable" key="egwck"></el-table-column>
                        <el-table-column width="40" prop="" class-name="no-text-tips" v-if="optDeviceShow" key="egwedit">
                            <template slot-scope="scope">
                                <div class="el-icon el-icon-operation-edit grayIcon" @click="modifyDevice(scope.row,event)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column prop="connection_status" width="50" sortable>
                                <template slot-scope="scope">
                                    
                                    <div :class="{
                                        'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
                                        '':scope.row.have_connected==2,
                                        'conn_exc':scope.row.connection_status=='Exception',
                                        'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
                                </template>
                        </el-table-column>
						<el-table-column v-if="detectable" label="<%=rb.getString("WeiZhiJianCe")%>" prop="isCheckInActive" width="160">
							<template slot-scope="scope">
								<div style="position:relative;">
									<el-switch :ref="scope.row.serial_number" v-model="scope.row.isCheckInActive" style="height: 18px;margin-left: 10px;"
										:disabled="true"
										active-color="#4D84FF"
										active-value="1"
										inactive-value="0">
									</el-switch>
									<div class="switch-cover" @click="setDetectEnable(scope.row)"></div>
								</div>
							</template>
						</el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' || deviceType == 'gNB'" key="device_status" label='<%=rb.getString("SheBeiZhuangTai")%>' min-width="60" width="105"  prop="device_status">
                            <template slot-scope="scope">
                                <!-- 图标在hover时显示状态文字 -->
                                <el-tooltip placement="right" :visible-arrow="false" popper-class="greenBg">
                                    <div slot="content" ><%=rb.getString("AnZhuang")%></div>
                                    <span  v-if="scope.row.device_status == '2'" class="el-icon el-icon-circle-success" style="font-size:22px ;color:#67c23a;"></span>
                                </el-tooltip>
                                <el-tooltip placement="right" :visible-arrow="false" popper-class="redBg">
                                    <div slot="content"><%=rb.getString("RuKu")%></div>
                                    <span  v-if="scope.row.device_status == '1'" class="el-icon el-icon-not-installed" style="font-size:22px ;color:#e88282;"></span>
                                </el-tooltip>
                                <el-tooltip placement="right" :visible-arrow="false" popper-class="redBg">
                                    <div slot="content"><%=rb.getString("BaoFei")%></div>
                                    <span  v-if="scope.row.device_status == '0'" class="el-icon el-icon-not-installed" style="font-size:22px ;color:#e88282;"></span>
                                </el-tooltip>
                            </template>
                        </el-table-column>
                        <el-table-column key="enbSn" v-if="deviceType == 'eNB'" label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="120" prop="serial_number"></el-table-column>
                        <el-table-column key="enbCellName" v-if="deviceType == 'eNB'" label='<%=rb.getString("HostName")%>' min-width="120" prop="host_name"></el-table-column>
                        <el-table-column key="gnbCellName" v-if="deviceType == 'gNB'" label='<%=rb.getString("GNBMingCheng")%>' min-width="120" prop="host_name"></el-table-column>
                        <el-table-column key="enbMac" v-if="deviceType == 'eNB'" label='<%=rb.getString("MACDiZhi")%>' min-width="120" prop="mac_address"></el-table-column>
                        <el-table-column key="enbGroupName" v-if="deviceType == 'eNB' || deviceType == 'gNB'" label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="100" prop="group_name"></el-table-column>
                        <el-table-column key="site_id" v-if="deviceType == 'eNB' && showSiteId == 'true'" label='<%=rb.getString("UPSSiteID")%>' min-width="100" prop="site_id"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol !=='true'" label='<%=rb.getString("ChangShang")%>' min-width="120" prop="manufacturer"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol !=='true'" label='<%=rb.getString("ChengShi")%>' min-width="100" prop="city"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol !=='true'" label='<%=rb.getString("SuoShuZhiJu")%>' min-width="120" prop="sub_branches"></el-table-column>
                        
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='<%=rb.getString("AnZhuangXiangXiDiZhi")%>' min-width="200" prop="install_address"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='<%=rb.getString("YeZhuLianXiFangShi")%>' min-width="100" prop="contact_number"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" :label="siteIdLabel" min-width="100" prop="site_id"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='Circuit Ref.' min-width="100" prop="circuit_ref"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='Circuit J & O' min-width="100" prop="circuit_jo"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='Status' min-width="100" prop="service_status"></el-table-column>
                        <el-table-column v-if="deviceType == 'eNB' && isDeviceMoreParams == 'true' && showOrHideCol =='true'" label='Rom' min-width="100" prop="rom"></el-table-column>
                        <el-table-column key="cpeSn" v-if="deviceType == 'CPE'" label='<%=rb.getString("CPEXuLieHao")%>' min-width="100" prop="serial_number"></el-table-column>
                        <el-table-column key="cpeMac" v-if="deviceType == 'CPE'" label='<%=rb.getString("MACDiZhi")%>' min-width="100" prop="macaddress"></el-table-column>
                        <el-table-column key="cpeImsi" v-if="deviceType == 'CPE'" label='IMSI' min-width="100" prop="imsi"></el-table-column>
                        <el-table-column key="longitude" v-if="deviceType == 'eNB' || deviceType == 'CPE' || deviceType == 'gNB'" label='<%=rb.getString("JingDu")%>' min-width="100" prop="longitude"></el-table-column>
                        <el-table-column key="latitude" v-if="deviceType == 'eNB' || deviceType == 'CPE' || deviceType == 'gNB'" label='<%=rb.getString("WeiDu")%>' min-width="100" prop="latitude"></el-table-column>
                        <el-table-column key="height" v-if="deviceType == 'eNB' || deviceType == 'CPE' || deviceType == 'gNB'" label='<%=rb.getString("GaoDu")%>' min-width="100" prop="height"></el-table-column>
                        <el-table-column key="cpeDistance" v-if="deviceType == 'CPE'" label='<%=rb.getString("JuLi")%>' min-width="100" prop="distance"></el-table-column>

                        <el-table-column key="gnbSn" v-if="deviceType == 'gNB'" label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100" prop="serial_number"></el-table-column>
                        <el-table-column key="gnbMac" v-if="deviceType == 'gNB'" label='<%=rb.getString("MACDiZhi")%>' min-width="100" prop="mac_address"></el-table-column>

                        <el-table-column v-if="['eNB','gNB'].includes(deviceType) && (isDeviceMoreParams == 'true' || (siteEnable && isDeviceMoreParams != 'true'))" :label="siteNameLabel" min-width="100" prop="sub_station_name"></el-table-column>

                        <el-table-column key="egwSn" v-if="deviceType == 'WCG'" prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
                        <el-table-column key="egwName" v-if="deviceType == 'WCG'" prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
                        <el-table-column key="egwIp" v-if="deviceType == 'WCG'" prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
                        <el-table-column key="egwPort" v-if="deviceType == 'WCG'" prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
                        <el-table-column key="peerIp" v-if="deviceType == 'WCG'" prop="peerIp" label="<%=rb.getString("HADuiDuanIP")%>" min-width="180"></el-table-column>
                        <!--<el-table-column prop="hardwareModel" label="<%=rb.getString("YingJianMoXing")%>" min-width="140"></el-table-column>-->
                        <el-table-column key="egwDescription" v-if="deviceType == 'WCG'" prop="egwDescription" label="<%=rb.getString("MiaoShu")%>" min-width="150"></el-table-column>
                        <el-table-column key="offlineDays" v-if="deviceType != 'WCG'" prop="offlineDays" label='<%=rb.getString("LiXianTianShu")%>' min-width="60" width="100"></el-table-column>

						<el-table-column v-if="deviceType == 'eNB'" prop="mechanical_downtilt" label="<%=rb.getString("JiXieXiaQingJiao")%>" min-width="180"></el-table-column>
						<el-table-column v-if="deviceType == 'eNB'" prop="electronic_downtilt" label="<%=rb.getString("DianZiXiaQingJiao")%>" min-width="180"></el-table-column>
						<el-table-column v-if="deviceType == 'eNB'" prop="vertical_3dB_beam_width" label="<%=rb.getString("ChuiZhiBoSuKuanDu")%>" min-width="180"></el-table-column>
						<el-table-column v-if="deviceType == 'eNB'" prop="horizontal_azimuth" label="<%=rb.getString("ShuiPinFangWeiJiao")%>" min-width="180"></el-table-column>
                        <el-table-column key="remark" v-if="deviceType != 'WCG' && deviceType != 'CPE'" prop="remark" width="185">
                            <template slot="header" slot-scope="scope">
                                <div class="remarkHeaderCls">
                                    <span v-if="!editingRemarkLabel">
                                        <span :title="currentRemarkLabel" style="max-width: 140px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: middle;">
                                            {{currentRemarkLabel}}
                                        </span>
                                        <i v-if="optDeviceShow" class="el-icon el-icon-operation-edit" @click="startEditRemarkLabel" style="margin-left: 5px;cursor: pointer;"></i>
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
                        </el-table-column>
                    </el-ctable>
                </div>
            </div>
            <div :class="rightBoxType == 'addGroup'? 'tablesListRightBoxCls addGroupCls' : 'tablesListRightBoxCls'" v-show="deviceListRightBoxShow">
                <div class="rightOutBoxHeadCls">
                    <span>{{rightBoxTitle}}</span>
                    <span class="el-icon el-icon-close greyIcon" @click="rightBoxClose"></span>
                </div>
                <div class="rightItemMainBox" v-if="rightBoxType == 'add'">
                    <el-form  :model="addDeviceForm" ref="addDeviceForm" :rules="addDeviceFormRules" label-position="top" id="addDeviceForm">
                        <el-form-item label="<%=rb.getString("TianJiaLeiXing") %>" prop="selectType" style='margin-bottom: 20px;'>
                            <el-radio-group v-model="addDeviceForm.selectType" >
                                <el-radio label="0" border size="small"><%=rb.getString("ShouDongShuRu") %></el-radio>
                                <el-radio v-if="batchOperation" label="1" border size="small"><%=rb.getString("PiLiangDaoRu") %></el-radio>
                            </el-radio-group>
                        </el-form-item>
                        <!--<el-form-item label="Network Element" prop="deviceType" style='margin-bottom: 20px;'>
                            <el-radio-group v-model="addDeviceForm.deviceType" @change="addDeviceTypeChange">
                                <el-radio v-for="item in deviceTypeList" :label="item.label" border size="small">{{item.label}}</el-radio-button>
                            </el-radio-group>
                        </el-form-item>-->
                        <el-form-item  prop='inputType' label="Input Type" style='margin-bottom:10px;' v-if="addDeviceForm.deviceType == 'CPE'">
                            <el-radio-group v-model="addDeviceForm.inputType" @change="addDeviceInputTypeChange">
                                <el-radio label="mac" border size="small">MAC</el-radio>
                                <el-radio label="sn" border size="small" style='margin-bottom:0px;'><%=rb.getString("CPEBianMa")%></el-radio>
                            </el-radio-group>
                        </el-form-item>
                        <div v-show="addDeviceForm.selectType == '0'" style="margin-bottom:10px;">
                            <div v-if="addDeviceForm.inputType == 'sn'">SN</div>
                            <div v-if="addDeviceForm.inputType == 'mac'">MAC</div>
                            <div v-if="isDeviceMoreParams != 'true' || addDeviceForm.deviceType != 'eNB'">
                                <el-form-item label='' prop='serialNumber' style='margin-bottom: 18px;'>
                                    <el-input type="textarea" v-model="addDeviceForm.serialNumber"></el-input>
                                </el-form-item>
                                <div class="tipText">
                                    <span class="el-icon el-icon-circle-info infoTip"></span>
                                    <span><%=rb.getString("cpeZhuCeTiShiWenZi")%></span>
                                </div>
                            </div>
                            <div v-if="isDeviceMoreParams == 'true' && addDeviceForm.deviceType == 'eNB'">
                                <el-form-item label='' prop='serialNumber' style='margin-bottom: 18px;'>
                                    <el-input v-model="addDeviceForm.serialNumber"></el-input>
                                </el-form-item>
                                <el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='group_id' style='margin-bottom: 20;'>
                                    <el-select v-model="addDeviceForm.group_id">
                                        <el-option v-for="item in deviceGroupOptions" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
                                    </el-select>
                                </el-form-item>
                                <div v-if='showOrHideCol !=="true"'>
                                    <el-form-item label="<%=rb.getString("ShengChanChangJia")%>" class="max-width-45" prop="manufacturer" required>
                                        <el-input v-model="addDeviceForm.manufacturer" maxLength="45"><el-input>
                                    </el-form-item> 
                                    <el-form-item label="<%=rb.getString("SheBeiXingHao")%>" class="max-width-45" prop="module_type" required>
                                        <el-input v-model="addDeviceForm.module_type" maxLength="45"><el-input>
                                    </el-form-item> 
                                    <el-form-item label="<%=rb.getString("HostName")%>" class="max-width-45" prop="host_name" required>
                                        <el-input v-model="addDeviceForm.host_name" maxLength="50"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("Sheng")%>" class="max-width-45" prop="province" required>
                                        <el-input v-model="addDeviceForm.province" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("ChengShi")%>" class="max-width-45" prop="city" required>
                                        <el-input v-model="addDeviceForm.city" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("XianQu")%>" class="max-width-45" prop="district" required>
                                        <el-input v-model="addDeviceForm.district" maxLength="128"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("XiangZhen")%>" class="max-width-45" prop="township" required>
                                        <el-input v-model="addDeviceForm.township" maxLength="128"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("SuoShuWangGe")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.sub_grid" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("SuoShuZhiJu")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.sub_branches" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("ZhanZhiBianMa")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.sub_station_code" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item v-show="['eNB', 'gNB'].includes(deviceType)" style="margin-top:15px;"
                                        v-if="siteEnable" 
                                        :label="siteNameLabel"
                                    >
                                        <el-select v-model="addDeviceForm.sub_station_name" popper-class="with-options" filterable>
                                            <el-option class="s-opts">
                                                <div @click="toAddSite">
                                                    <div class="option-bt">Create New Site</div>
                                                </div>
                                            </el-option>
                                            <el-option v-for="item in siteNames" :label="item.siteName" :value="item.siteName"></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item v-if="!siteEnable"  :label="siteNameLabel" class="max-width-45">
                                        <el-input v-model="addDeviceForm.sub_station_name" maxLength="64"><el-input>
                                    </el-form-item>
									<!-- topo part -->
									<el-form-item prop="mechanical_downtilt" label="<%=rb.getString("JiXieXiaQingJiao")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.mechanical_downtilt" maxLength="5" placeholder="Range: 0-9"><el-input>
                                    </el-form-item>
									<el-form-item prop="vertical_3dB_beam_width" label="<%=rb.getString("ChuiZhiBoSuKuanDu")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.vertical_3dB_beam_width" maxLength="5" placeholder="Range: 1-9"><el-input>
                                    </el-form-item>
									<el-form-item prop="horizontal_azimuth" label="<%=rb.getString("ShuiPinFangWeiJiao")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.horizontal_azimuth" maxLength="5" placeholder="Range: 0-359"><el-input>
                                    </el-form-item>

                                    <el-form-item label="<%=rb.getString("FenBuXiTong")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.sub_distribute_system" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("ShouDongLuRuECI")%>" class="max-width-45" prop="sub_cell_id" required>
                                        <el-input v-model="addDeviceForm.sub_cell_id" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("SuoShuENBMingCheng")%>" class="max-width-45" prop="sub_cell_name" required>
                                        <el-input v-model="addDeviceForm.sub_cell_name" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("SuoShuTAList")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.sub_talist" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("FuGaiChangJingShuXing")%>" class="max-width-45" prop="over_scen_attr" required>
                                        <el-select v-model='addDeviceForm.over_scen_attr'>
                                            <el-option label='<%=rb.getString("PuTongYongHu")%>' value='1'></el-option>
                                            <el-option label='<%=rb.getString("DianTiDiXiaShi")%>' value='2'></el-option>
                                            <el-option label='<%=rb.getString("ShiNeiFenBuXiTong")%>' value="3"></el-option>
                                            <el-option label='<%=rb.getString("QiTa")%>' value="4"></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("SheBeiGongLv")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.maxtxpower" maxLength="45"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("SheBeiJieRuFangShi")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.device_access_mode" maxLength="32"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("RXDuanKouShu")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.rx_port_number" maxLength="11"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("TXDuanKouShu")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.tx_port_number" maxLength="11"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("WangGuanIP")%>" prop="omc_ip" class="max-width-45" required>
                                        <el-input v-model="addDeviceForm.omc_ip" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("SheBeiIP")%>" prop="cell_ip" class="max-width-45" required>
                                        <el-input v-model="addDeviceForm.cell_ip" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("TAC")%>" prop="tac" class="max-width-45" required>
                                        <el-input v-model="addDeviceForm.tac" maxLength="200"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("RuanJianBanBen")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.software_version" maxLength="500"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("WangYuanDengJi")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.net_grade" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("AnZhuangJingDu")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.gps_longitude" maxLength="16"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("AnZhuangWeiDu")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.gps_latitude" maxLength="16"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("AnZhuangXiangXiDiZhi")%>" class="max-width-45" prop="install_address" required>
                                        <el-input v-model="addDeviceForm.install_address" maxLength="256"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("RuWangShiJian")%>" class="max-width-45">
                                        <el-date-picker v-model="addDeviceForm.access_net_date" type="datetime" :editable="false" value-format="yyyy-MM-dd HH:mm:ss"></el-date-picker>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("DaiWeiDuiWuHao")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.agent_maintain" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("YeZhuLianXiRen")%>" class="max-width-45" prop="contact_person" required>
                                        <el-input v-model="addDeviceForm.contact_person" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("YeZhuLianXiFangShi")%>" class="max-width-45" prop="contact_number" required>
                                        <el-input v-model="addDeviceForm.contact_number" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("SheBeiZhuangTai")%>" class="max-width-45">
                                        <el-select v-model='addDeviceForm.device_status'>
                                            <el-option label='<%=rb.getString("RuKu")%>' :value='1'></el-option>
                                            <el-option label='<%=rb.getString("AnZhuang")%>' :value='2'></el-option>
                                            <el-option label='<%=rb.getString("BaoFei")%>' :value="0"></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("ShangLianKuanDaiZhangHu")%>" class="max-width-45" prop="uplink_broadband_account" required>
                                        <el-input v-model="addDeviceForm.uplink_broadband_account" maxLength="64"><el-input>
                                    </el-form-item>
                                </div>
                                <div v-show="showOrHideCol == 'true'">
                                    <el-form-item style="margin-top:15px;" v-show="['eNB', 'gNB'].includes(deviceType)"
                                        v-if="siteEnable" 
                                        :label="siteNameLabel"
                                    >
                                        <el-select v-model="addDeviceForm.sub_station_name" popper-class="with-options" filterable>
                                            <el-option class="s-opts">
                                                <div @click="toAddSite">
                                                    <div class="option-bt">Create New Site</div>
                                                </div>
                                            </el-option>
                                            <el-option v-for="item in siteNames" :label="item.siteName" :value="item.siteName"></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item v-if="!siteEnable" :label="siteNameLabel" class="max-width-45">
                                        <el-input v-model="addDeviceForm.sub_station_name" maxLength="64"><el-input>
                                    </el-form-item>
									<!-- topo part -->
									<el-form-item prop="mechanical_downtilt" label="<%=rb.getString("JiXieXiaQingJiao")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.mechanical_downtilt" maxLength="5" placeholder="Range: 0-9"><el-input>
                                    </el-form-item>
									<el-form-item prop="vertical_3dB_beam_width" label="<%=rb.getString("ChuiZhiBoSuKuanDu")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.vertical_3dB_beam_width" maxLength="5" placeholder="Range: 1-9"><el-input>
                                    </el-form-item>
									<el-form-item prop="horizontal_azimuth" label="<%=rb.getString("ShuiPinFangWeiJiao")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.horizontal_azimuth" maxLength="5" placeholder="Range: 0-359"><el-input>
                                    </el-form-item>
									
                                    <el-form-item label="<%=rb.getString("AnZhuangXiangXiDiZhi")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.install_address" maxLength="256"><el-input>
                                    </el-form-item>
                                    <el-form-item label="<%=rb.getString("YeZhuLianXiFangShi")%>" class="max-width-45">
                                        <el-input v-model="addDeviceForm.contact_number" maxLength="64"><el-input>
                                    </el-form-item>
                                    <el-form-item :label="siteIdLabel" class="max-width-45">
                                        <el-input v-model="addDeviceForm.site_id" maxLength="256"><el-input>
                                    </el-form-item>
                                    <el-form-item label="Circuit Ref." class="max-width-45">
                                        <el-input v-model="addDeviceForm.circuit_ref" maxLength="128"><el-input>
                                    </el-form-item>
                                    <el-form-item label="Circuit J & O" class="max-width-45">
                                        <el-input v-model="addDeviceForm.circuit_jo" maxLength="128"><el-input>
                                    </el-form-item>
                                    <el-form-item label="Status" class="max-width-45">
                                        <el-input v-model="addDeviceForm.service_status" maxLength="50"><el-input>
                                    </el-form-item>
                                    <el-form-item label="Rom" class="max-width-45">
                                        <el-input v-model="addDeviceForm.rom" maxLength="50"><el-input>
                                    </el-form-item>
                                </div>
                            </div>
                        </div>
                        <div v-show="addDeviceForm.selectType == '1'" style="margin-bottom:20px;">
                            <div style="display:flex;" class="uploadBox">
                                <el-form-item label='' prop='' style="margin-bottom: 0px;" class='importFileItem'>
                                    <template slot="label">
                                        <%=rb.getString("DaoRuWenJian")%>
                                        <span style="color:#7A7992">(<%=rb.getString("DaoRuWenJianLeiXing")%>)</span>
                                    </template>
                                    <el-upload ref="addUpload"
                                        :before-upload='addBeforeUpload' 
                                        :on-success='addCheckFile' 
                                        :on-change="addFileChange" 
                                        :show-file-list="false" 
                                        :action="addUploadFileURL" 
                                        :data="addFileParams" 
                                        name="uploadFile" 
                                        :auto-upload="false"
                                        accept=".xlsx, .csv">
                                        <el-input :readonly="true" :value="addFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:300px;">
                                            <a slot="append" class="el-icon el-icon-operation-import grayIcon" @click="addFileSelect"></a>
                                        </el-input>
                                        <div slot="tip" class="el-upload__tip" v-show="!typeFlag"><%=rb.getString("ZhiZhiChiXLSXCSVWenJian")%></div>
                                        <div slot="tip" class="el-upload__tip" v-show="addSelectFlag"><%=rb.getString("QingXianXuanZeWenJian")%></div>
                                        <a slot="trigger" ref="addFile_up"></a>
                                    </el-upload>	
                                </el-form-item>
                            </div>
                            <div style='margin-top: 12px;'>
                                <div class="tipText">
                                    <span class="el-icon el-icon-circle-info infoTip"></span>
                                    <span><%=rb.getString("DaoRuWenJianTiShi")%></span>
                                </div>
                                <div style="cursor:pointer;padding-top:10px;" @click="exportAddTemplate">
                                    <span class='el-icon el-icon-common-download exportTemplateIcon'></span>
                                    <span class='exportTemplateText'><%=rb.getString("DaoChuMuBan")%></span>
                                </div>
                            </div>
                        </div>
                        <el-form-item style='margin-bottom: 0;'
                            v-if="(isDeviceMoreParams != 'true' || addDeviceForm.deviceType != 'eNB') || addDeviceForm.selectType == '1'" 
                            label='<%=rb.getString("SheBeiZuMingCheng")%>' 
                            prop='group_id'
                        >
                            <el-select v-model="addDeviceForm.group_id">
                                <el-option v-for="item in deviceGroupOptions" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
                            </el-select>
                        </el-form-item>
                        <el-form-item style="margin-top:15px;" v-show="['eNB', 'gNB'].includes(deviceType)"
                            v-if="siteEnable && isDeviceMoreParams != 'true' && addDeviceForm.selectType == '0' || (siteEnable && addDeviceForm.deviceType == 'gNB' && addDeviceForm.selectType == '0')" 
                            :label="siteNameLabel"
                            prop="sub_station_name"
                        >
                            <el-select v-model="addDeviceForm.sub_station_name" popper-class="with-options" filterable>
                                <el-option class="s-opts">
                                    <div @click="toAddSite">
                                        <div class="option-bt">Create New Site</div>
                                    </div>
                                </el-option>
                                <el-option v-for="item in siteNames" :label="item.siteName" :value="item.siteName"></el-option>
                            </el-select>
                        </el-form-item>
                    </el-form>
                </div>
                <div class="rightItemMainBox" v-if="rightBoxType == 'modify'">
                    <el-form  :model="modifyDeviceForm" ref="modifyDeviceForm" :rules="modifyDeviceFormRules" label-position="top" id="modifyDeviceForm">
                        <div v-show="deviceType == 'eNB' || deviceType == 'CPE'">
                            <el-form-item v-show="deviceType == 'eNB'" label='<%=rb.getString("XiaoZhanBianMa")%>'>
                                <el-input v-model="modifyDeviceForm.serial_number" :disabled="true"></el-input>
                            </el-form-item>
                            <el-form-item v-show="deviceType == 'CPE'" label='<%=rb.getString("MACDiZhi")%>'>
                                <el-input v-model="modifyDeviceForm.macaddress" :disabled="true"></el-input>
                            </el-form-item>
                            <el-form-item label="<%=rb.getString("JingDu")%>" prop="longitude">
                                <el-input v-model="modifyDeviceForm.longitude"></el-input>
                            </el-form-item>
                            <el-form-item label="<%=rb.getString("WeiDu")%>" prop="latitude">
                                <el-input v-model="modifyDeviceForm.latitude"></el-input>
                            </el-form-item>
                            <el-form-item label="<%=rb.getString("GaoDu")%>" prop="height">
                                <el-input v-model="modifyDeviceForm.height"></el-input>
                            </el-form-item>
                            <el-form-item v-show="deviceType == 'CPE'" label="<%=rb.getString("JuLi")%>" prop="distance">
                                <el-input v-model="modifyDeviceForm.distance"></el-input>
                            </el-form-item>
							<el-form-item v-show="deviceType == 'eNB' && detectable" prop="isCheckInActive">
								<div slot="label" style="display: flex;align-items: center;">
									<%=rb.getString("WeiZhiJianCe")%>
									<el-tooltip placement="bottom">
										<div slot="content">
											<div><%=rb.getString("WeiZhiJianCeChongQiTi")%></div>
										</div>
										<div class="promptInfo" ><span class="el-icon-circle-info el-icon"></span></div>
									</el-tooltip>
								</div>
								<el-switch v-model="modifyDeviceForm.isCheckInActive" style="height: 18px;margin-left: 10px;"
									:disabled="false"
									active-color="#4D84FF"
									active-value="1"
									inactive-value="0">
								</el-switch>
							</el-form-item>
                            <el-form-item v-show="deviceType == 'eNB'" label="<%=rb.getString("SheBeiZhuangTai")%>"  prop="device_status">
                                <el-select v-model='modifyDeviceForm.device_status'>
                                    <el-option label='<%=rb.getString("RuKu")%>' :value='1'></el-option>
                                    <el-option label='<%=rb.getString("AnZhuang")%>' :value='2'></el-option>
                                    <el-option label='<%=rb.getString("BaoFei")%>' :value="0"></el-option>
                                </el-select>
                            </el-form-item>
                            
                            <el-form-item v-show="['eNB', 'gNB'].includes(deviceType)" style="margin-top:15px;" class="max-width-45"
                                v-if="siteEnable" 
                                :label="siteNameLabel"
                                prop="sub_station_name"
                            >
                                <el-select v-model="modifyDeviceForm.sub_station_name" popper-class="with-options" filterable>
                                    <el-option class="s-opts">
                                        <div @click="toAddSite">
                                            <div class="option-bt">Create New Site</div>
                                        </div>
                                    </el-option>
                                    <el-option v-for="item in siteNames" :label="item.siteName" :value="item.siteName"></el-option>
                                </el-select>
                            </el-form-item>
							<!-- topo part -->
							<el-form-item v-show="deviceType == 'eNB'" prop="mechanical_downtilt" label="<%=rb.getString("JiXieXiaQingJiao")%>" class="max-width-45">
								<el-input v-model="modifyDeviceForm.mechanical_downtilt" maxLength="5" placeholder="Range: 0-9"><el-input>
							</el-form-item>
							<el-form-item v-show="deviceType == 'eNB'" prop="vertical_3dB_beam_width" label="<%=rb.getString("ChuiZhiBoSuKuanDu")%>" class="max-width-45">
								<el-input v-model="modifyDeviceForm.vertical_3dB_beam_width" maxLength="5" placeholder="Range: 1-9"><el-input>
							</el-form-item>
							<el-form-item v-show="deviceType == 'eNB'" prop="horizontal_azimuth" label="<%=rb.getString("ShuiPinFangWeiJiao")%>" class="max-width-45">
								<el-input v-model="modifyDeviceForm.horizontal_azimuth" maxLength="5" placeholder="Range: 0-359"><el-input>
							</el-form-item>
                            <el-form-item v-show="deviceType == 'eNB' && showSiteId == 'true'" prop="site_id" :label="siteIdLabel" class="max-width-45">
                                <el-input v-model="modifyDeviceForm.site_id" maxLength="256"><el-input>
                            </el-form-item>
                        </div>
                        <div v-show="deviceType == 'gNB'">
                            <el-form-item  label='<%=rb.getString("XiaoZhanBianMa")%>'>
                                <el-input v-model="modifyDeviceForm.serial_number" :disabled="true"></el-input>
                            </el-form-item>
                            <el-form-item label="<%=rb.getString("JingDu")%>" prop="longitude">
                                <el-input v-model="modifyDeviceForm.longitude"></el-input>
                            </el-form-item>
                            <el-form-item label="<%=rb.getString("WeiDu")%>" prop="latitude">
                                <el-input v-model="modifyDeviceForm.latitude"></el-input>
                            </el-form-item>
                            <el-form-item label="<%=rb.getString("GaoDu")%>" prop="height">
                                <el-input v-model="modifyDeviceForm.height"></el-input>
                            </el-form-item>
                            
                            <el-form-item label="<%=rb.getString("SheBeiZhuangTai")%>"  prop="device_status">
                                <el-select v-model='modifyDeviceForm.device_status'>
                                    <el-option label='<%=rb.getString("RuKu")%>' :value='1'></el-option>
                                    <el-option label='<%=rb.getString("AnZhuang")%>' :value='2'></el-option>
                                    <el-option label='<%=rb.getString("BaoFei")%>' :value="0"></el-option>
                                </el-select>
                            </el-form-item>

                            <el-form-item style="margin-top:15px;" class="max-width-45"
                                v-if="siteEnable" 
                                :label="siteNameLabel"
                                prop="sub_station_name"
                            >
                                <el-select v-model="modifyDeviceForm.sub_station_name" popper-class="with-options" filterable>
                                    <el-option class="s-opts">
                                        <div @click="toAddSite">
                                            <div class="option-bt">Create New Site</div>
                                        </div>
                                    </el-option>
                                    <el-option v-for="item in siteNames" :label="item.siteName" :value="item.siteName"></el-option>
                                </el-select>
                            </el-form-item>
                        </div>
                        <div v-show="deviceType == 'WCG'">
                            <el-form-item label='<%=rb.getString("XiaoZhanBianMa")%>'>
                                <el-input v-model="modifyDeviceForm.egwSn" :disabled="true"></el-input>
                            </el-form-item>
                            <el-form-item label="<%=rb.getString("EGWMingCheng")%>" prop='egwName'>
                                <el-input v-model.trim='modifyDeviceForm.egwName' maxlength="32"></el-input>
                            </el-form-item>
                            <el-form-item label="<%=rb.getString("YingJianMoXing")%>" prop='hardwareModel'>
                                <el-input v-model.trim='modifyDeviceForm.hardwareModel' maxlength="50"></el-input>
                            </el-form-item>
                            <el-form-item prop="egwDescription" label="<%=rb.getString("MiaoShu")%>">
                                <el-input type="textarea" resize="true" :rows="3" maxlength="500" v-model="modifyDeviceForm.egwDescription"></el-input>
                            </el-form-item>
                        </div>
                        <div v-if="isDeviceMoreParams == 'true' && deviceType == 'eNB'">
                            <div v-if='showOrHideCol !=="true"'>
                                <el-form-item prop="manufacturer" label="<%=rb.getString("ShengChanChangJia")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.manufacturer" maxLength="45"><el-input>
                                </el-form-item> 
                                <el-form-item prop="module_type" label="<%=rb.getString("SheBeiXingHao")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.module_type" maxLength="45"><el-input>
                                </el-form-item> 
                                <el-form-item prop="host_name" label="<%=rb.getString("HostName")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.host_name" maxLength="50"><el-input>
                                </el-form-item>
                                <el-form-item prop="province" label="<%=rb.getString("Sheng")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.province" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="city" label="<%=rb.getString("ChengShi")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.city" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="district" label="<%=rb.getString("XianQu")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.district" maxLength="128"><el-input>
                                </el-form-item>
                                <el-form-item prop="township" label="<%=rb.getString("XiangZhen")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.township" maxLength="128"><el-input>
                                </el-form-item>
                                <el-form-item prop="sub_grid" label="<%=rb.getString("SuoShuWangGe")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.sub_grid" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="sub_branches" label="<%=rb.getString("SuoShuZhiJu")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.sub_branches" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="sub_station_code" label="<%=rb.getString("ZhanZhiBianMa")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.sub_station_code" maxLength="64"><el-input>
                                </el-form-item>
                                
                                <el-form-item v-if="!siteEnable" prop="sub_station_name" :label="siteNameLabel" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.sub_station_name" maxLength="64"><el-input>
                                </el-form-item>
								
                                <el-form-item prop="sub_distribute_system" label="<%=rb.getString("FenBuXiTong")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.sub_distribute_system" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="sub_cell_id" label="<%=rb.getString("ShouDongLuRuECI")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.sub_cell_id" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="sub_cell_name" label="<%=rb.getString("SuoShuENBMingCheng")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.sub_cell_name" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="sub_talist" label="<%=rb.getString("SuoShuTAList")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.sub_talist" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="over_scen_attr" label="<%=rb.getString("FuGaiChangJingShuXing")%>" class="max-width-45" required>
                                    <el-select v-model='modifyDeviceForm.over_scen_attr'>
                                        <el-option label='<%=rb.getString("PuTongYongHu")%>' value='1'></el-option>
                                        <el-option label='<%=rb.getString("DianTiDiXiaShi")%>' value='2'></el-option>
                                        <el-option label='<%=rb.getString("ShiNeiFenBuXiTong")%>' value="3"></el-option>
                                        <el-option label='<%=rb.getString("QiTa")%>' value="4"></el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item prop="maxtxpower" label="<%=rb.getString("SheBeiGongLv")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.maxtxpower" maxLength="45"><el-input>
                                </el-form-item>
                                <el-form-item prop="device_access_mode" label="<%=rb.getString("SheBeiJieRuFangShi")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.device_access_mode" maxLength="32"><el-input>
                                </el-form-item>
                                <el-form-item prop="rx_port_number" label="<%=rb.getString("RXDuanKouShu")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.rx_port_number" maxLength="11"><el-input>
                                </el-form-item>
                                <el-form-item prop="tx_port_number" label="<%=rb.getString("TXDuanKouShu")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.tx_port_number" maxLength="11"><el-input>
                                </el-form-item>
                                <el-form-item prop="omc_ip" label="<%=rb.getString("WangGuanIP")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.omc_ip" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="cell_ip" label="<%=rb.getString("SheBeiIP")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.cell_ip" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="tac" label="<%=rb.getString("TAC")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.tac" maxLength="200"><el-input>
                                </el-form-item>
                                <el-form-item prop="software_version" label="<%=rb.getString("RuanJianBanBen")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.software_version" maxLength="500"><el-input>
                                </el-form-item>
                                <el-form-item prop="net_grade" label="<%=rb.getString("WangYuanDengJi")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.net_grade" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="install_address" label="<%=rb.getString("AnZhuangXiangXiDiZhi")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.install_address" maxLength="256"><el-input>
                                </el-form-item>
                                <el-form-item prop="access_net_date" label="<%=rb.getString("RuWangShiJian")%>" class="max-width-45">
                                    <el-date-picker v-model="modifyDeviceForm.access_net_date" type="datetime" :editable="false" value-format="yyyy-MM-dd HH:mm:ss"></el-date-picker>
                                </el-form-item>
                                <el-form-item prop="agent_maintain" label="<%=rb.getString("DaiWeiDuiWuHao")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.agent_maintain" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="contact_person" label="<%=rb.getString("YeZhuLianXiRen")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.contact_person" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="contact_number" label="<%=rb.getString("YeZhuLianXiFangShi")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.contact_number" maxLength="64"><el-input>
                                </el-form-item>
                                
                                <el-form-item prop="uplink_broadband_account" label="<%=rb.getString("ShangLianKuanDaiZhangHu")%>" class="max-width-45" required>
                                    <el-input v-model="modifyDeviceForm.uplink_broadband_account" maxLength="64"><el-input>
                                </el-form-item>
                            </div>
                            <div v-show="showOrHideCol == 'true'">
                                <el-form-item prop="host_name" label="<%=rb.getString("HostName")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.host_name" maxLength="50"><el-input>
                                </el-form-item>

                                <el-form-item v-if="!siteEnable" prop="sub_station_name" :label="siteNameLabel" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.sub_station_name" maxLength="64"><el-input>
                                </el-form-item>

                                <el-form-item prop="install_address" label="<%=rb.getString("AnZhuangXiangXiDiZhi")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.install_address" maxLength="256"><el-input>
                                </el-form-item>
                                <el-form-item prop="contact_number" label="<%=rb.getString("YeZhuLianXiFangShi")%>" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.contact_number" maxLength="64"><el-input>
                                </el-form-item>
                                <el-form-item prop="site_id" :label="siteIdLabel" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.site_id" maxLength="256"><el-input>
                                </el-form-item>
                                <el-form-item prop="circuit_ref" label="Circuit Ref." class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.circuit_ref" maxLength="128"><el-input>
                                </el-form-item>
                                <el-form-item prop="circuit_jo" label="Circuit J & O" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.circuit_jo" maxLength="128"><el-input>
                                </el-form-item>
                                <el-form-item prop="service_status" label="Status" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.service_status" maxLength="50"><el-input>
                                </el-form-item>
                                <el-form-item prop="rom" label="Rom" class="max-width-45">
                                    <el-input v-model="modifyDeviceForm.rom" maxLength="50"><el-input>
                                </el-form-item>
                            </div>
                        </div>
                        <div v-show="deviceType == 'eNB' || deviceType == 'gNB'">
                            <el-form-item :label='currentRemarkLabel' prop="remark">
                                <el-input v-model="modifyDeviceForm.remark" maxlength="30"></el-input>
                            </el-form-item>
                        </div>
                    </el-form>
                </div>
                <div class="rightItemMainBox" v-if="rightBoxType == 'addGroup'">
                    <el-form  :model="addGroupForm" ref="addGroupForm" :rules="addGroupFormRules" label-position="top" id="addGroupForm">
                        <div v-show="addGroupType == 'addOneGroup'">
                            <el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='topGroupName' style='margin-bottom: 18px;'>
                                <el-input v-model="addGroupForm.topGroupName" maxlength="50"></el-input>
                            </el-form-item>
                            <div style="margin-bottom:20px;">
                                <el-checkbox v-model="addGroupForm.addSubCheck"></el-checkbox>
                                <span style="margin-left:5px;">add a sub-device group</span> 
                                <span style="display:block;color:rgba(0, 0, 0, 0.4);margin-left:20px;"><%=rb.getString("YiJiSheBeiZuGouXuanTianJiaErJiTiShi")%></span>
                            </div>
                        </div>
                        <div v-show="addGroupForm.addSubCheck">
                            <el-form-item label='Subgroup Name' prop='subGroupName' style='margin-bottom: 18px;'>
                                <el-input v-model="addGroupForm.subGroupName" maxlength="50" :disabled="addGroupType == 'viewSubGroup'"></el-input>
                            </el-form-item>
                            <el-form-item v-if="deviceType !== 'WCG'" label="<%=rb.getString("ZiDongFenPeiDaoSheBeiZu")%>" prop="enable" style='margin-bottom: 20px;'>
                                <el-switch v-model="addGroupForm.enable" active-value="1" inactive-value="0" :disabled="addGroupType == 'viewSubGroup'"></el-switch>
                            </el-form-item>
                            <div v-if="deviceType !== 'WCG' && addGroupForm.enable == '1'">
                                <el-form-item prop="matching_mode" style='margin-bottom: 20px;'>
                                    <span slot="label">
                                        <%=rb.getString("PiPeiGuiZe")%>
                                        <span v-if="addGroupForm.matching_mode == 'deviceName'" style="color: rgba(0,0,0,0.32);">（<%=rb.getString("BuChaoGuo")%> 10）</span>
                                    </span>
                                    <el-radio-group v-model="addGroupForm.matching_mode" @change="matchingModeChange">
                                        <el-radio label="deviceName" border size="small"><%=rb.getString("Title_SheBeiMingCheng")%></el-radio>
                                        <el-radio v-if="isSupportGSM && deviceType == 'eNB'" label="lac" border size="small">LAC</el-radio>
                                        <el-radio v-if="deviceType !== 'CPE'" label="tac" border size="small">TAC</el-radio>
                                    </el-radio-group>
                                </el-form-item>
                                   
                                <el-form-item v-if="addGroupForm.matching_mode == 'deviceName'" style='margin-bottom: 0px;'>
                                    <div v-for="(filter, index) in nameContainsContentList" :key="index" class="nameContainsContentBoxCls">
                                        <!-- 第一行：只有 filterCondition 下拉和输入框 -->
                                        <div v-if="index === 0" style="display: flex; align-items: center; gap: 5px;">
                                            <el-select v-model="filter.condition" style="width: 120px;">
                                                <el-option 
                                                    v-for="option in filterConditionOptions" 
                                                    :key="option.value"
                                                    :label="option.label" 
                                                    :value="option.value">
                                                </el-option>
                                            </el-select>
                                            <el-input 
                                                v-model="filter.value" 
                                                style="width: 325px;"
                                                :class="{'input-error': filter.hasError}"
                                                maxlength="64"
                                                @blur="validateSingleInput(index)"
                                                @input="validateSingleInput(index)">
                                            </el-input>
                                        </div>
                                        <!-- 其他行：And/Or 下拉、filterCondition 下拉和输入框 -->
                                        <div v-else style="display: flex; align-items: center; gap: 5px;">
                                            <el-select v-model="filter.andOr" style="width: 70px;">
                                                <el-option 
                                                    v-for="option in getAndOrOptions(index)" 
                                                    :key="option.value"
                                                    :label="option.label" 
                                                    :value="option.value"
                                                    :disabled="option.disabled">
                                                </el-option>
                                            </el-select>
                                            <el-select v-model="filter.condition" style="width: 120px;">
                                                <el-option 
                                                    v-for="option in filterConditionOptions" 
                                                    :key="option.value"
                                                    :label="option.label" 
                                                    :value="option.value">
                                                </el-option>
                                            </el-select>
                                            <el-input 
                                                v-model="filter.value" 
                                                style="width: 225px;"
                                                :class="{'input-error': filter.hasError}"
                                                maxlength="64"
                                                @blur="validateSingleInput(index)"
                                                @input="validateSingleInput(index)">
                                            </el-input>
                                            <span @click="removeFilter(index)" style="cursor: pointer;" class="el-icon el-icon-circle-close"></span>
                                        </div>
                                    </div>
                                    <!-- 新增按钮放在列表下方 -->
                                    <div @click="addNameContainsContent" v-if="nameContainsContentList.length < 10" class="addNameContainsContentBoxCls" style="margin-top: 10px;">
                                        <span class="el-icon el-icon-plus" style="margin-right: 5px;"></span>
                                        <span><%=rb.getString("TianJiaTiaoJian")%></span>
                                    </div>
                                </el-form-item>
                                <el-form-item v-if="addGroupForm.matching_mode == 'deviceName'" prop="name_contains" style='margin-bottom: 20px;'>
                                    <el-input v-model="addGroupForm.name_contains" v-show="false"></el-input>
                                </el-form-item>
                                <div v-if="nowInputNameContainsContent" class="nowInputNameContainsContentBoxCls">{{nowInputNameContainsContent}}</div>
                                <el-form-item v-if="addGroupForm.matching_mode == 'tac' || addGroupForm.matching_mode == 'lac'" :label="addGroupForm.matching_mode == 'lac' ? 'LAC': 'TAC'" prop="tac_rag" style='margin-bottom: 20px;'>
                                    <el-input v-model="addGroupForm.tac_rag" maxlength="50" placeholder="eg: 1,2,3,1-3"></el-input>
                                </el-form-item>
                            </div>
                            <el-form-item label='<%=rb.getString("MiaoShu")%>' prop='desc'>
                                <el-input type="textarea" maxlength="50" v-model="addGroupForm.desc" :disabled="addGroupType == 'viewSubGroup'"></el-input>
                            </el-form-item>
                            <!--<el-form-item label="Network Element" prop="deviceType" style='margin-bottom: 20px;'>
                                <el-checkbox-group v-model="addGroupForm.deviceType" @change="addGroupDeviceTypeChange" :disabled="true">
                                    <el-checkbox v-for="item in deviceTypeList" :label="item.label" border size="small">{{item.label}}</el-checkbox>
                                </el-checkbox-group>
                            </el-form-item>-->
                            <div class="addGroupDeviceBoxCls" v-show="addGroupForm.deviceType.length >0" :class="addGroupType == 'addOneGroup' ? '' : 'tableCalss'">
                                <el-tabs class="fit newTabs" v-model="addGroupDeviceActiveName" :editable="addGroupType == 'viewSubGroup'? false : false" style='height:100%' @edit="addGroupDeviceTabsEdit">
                                    <el-tab-pane v-if="item.isShow" :key="item.label" v-for="item in addGroupDeviceList" :label="item.label" :name="item.label" :key="item.label">
                                        <el-pairgrid :readonly="addGroupType == 'viewSubGroup'? true : false" :id="item.refStr" :ref="item.refStr" :key="item.refStr" @selection-change='addGroupSelectChange($event,item.label)'  
                                            :right-url="item.rightUrl" :left-url="item.leftUrl" :height="height" :row-key="item.rowKey" :query-params="item.queryParams" :title="item.title" 
                                            :messages="item.messages">
                                                <template slot="left">
                                                    <el-table-column type="selection" width="45"></el-table-column>
                                                    <el-table-column prop="connection_status" width="50">
                                                        <template slot-scope="scope">
                                                            <div :class="{
                                                                'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
                                                                '':scope.row.have_connected==2,
                                                                'conn_exc':scope.row.connection_status=='Exception',
                                                                'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
                                                        </template>
                                                    </el-table-column>
                                                    <el-table-column key="enbSn" v-if="item.label == 'eNB'" prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="150"></el-table-column>
                                                    <el-table-column key="enbGroup" v-if="item.label == 'eNB'" prop='group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="150"></el-table-column>
                                                    
                                                    <el-table-column key="cpeSn" v-if="item.label == 'CPE'" prop='serial_number' label='<%=rb.getString("CPEBianMa")%>' min-width="150"></el-table-column>
                                                    <el-table-column key="cpeMac" v-if="item.label == 'CPE'" prop='macaddress' label='<%=rb.getString("MACDiZhi")%>' min-width="150"></el-table-column>
                                                    <el-table-column key="cpeImsi" v-if="item.label == 'CPE'" prop='imsi' label='IMSI' min-width="80"></el-table-column>
                                                    <el-table-column key="cpeGroup" v-if="item.label == 'CPE'" prop='device_group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="150"></el-table-column>
                                                    
                                                    <el-table-column key="gnbSn" v-if="item.label == 'gNB'" label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100" prop="serial_number" min-width="150"></el-table-column>
                                                    <el-table-column key="gnbGroup" v-if="item.label == 'gNB'"  prop='group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="150"></el-table-column>
                                                    
                                                    <el-table-column key="egwSn" v-if="item.label == 'WCG'" prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
                                                
                                                </template>
                                                <template slot='toolbar'>
                                                    <div style="padding-left:10px;display:flex;">
                                                        <el-select v-model="item.queryParams.group_id" v-if="item.label != 'WCG'">
                                                            <el-option v-for="items in addGroupOptions" :key="items.id" :label="items.group_name" :value="items.id"></el-option>
                                                        </el-select>
                                                        <div class="queryGroup">
                                                            <el-input v-model="item.search_text" @keyup.enter.native="addGroupTableQuery(item.label)" :placeholder='item.messages.placeholder' style="width:180px;"></el-input>
                                                            <i @click='addGroupTableQuery(item.label)' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
                                                        </div>
                                                    </div>
                                                </template>
                                                <template slot='right'>
                                                    <el-table-column key="enbSn" v-if="item.label == 'eNB'" prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100"></el-table-column>
                                                    <el-table-column key="cpeSn" v-if="item.label == 'CPE'" prop='serial_number' label='<%=rb.getString("CPEBianMa")%>' min-width="100"></el-table-column>
                                                    <el-table-column key="cpeMac" v-if="item.label == 'CPE'" prop='macaddress' label='<%=rb.getString("MACDiZhi")%>' min-width="100"></el-table-column>
                                                    <el-table-column key="gnbSn" v-if="item.label == 'gNB'" label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100" prop="serial_number"></el-table-column>
                                                    <el-table-column key="egwSn" v-if="item.label == 'WCG'" prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
                                            </template>
                                        </el-pairgrid>
                                    </el-tab-pane>
                                </el-tabs>
                            </div>
                        </div>
                    </el-form>
                </div>
                <div class="footer" v-if="addGroupType !== 'viewSubGroup'">
                    <div class="lnkbuttonGroup" v-show="rightBoxType == 'add'" style="margin-left:20px;">
                        <el-button  type="primary"  @click="addDeviceSubmit" :loading="showSubmitLoading"><%=rb.getString("QueDing")%></el-button>
                        <el-button  @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
                    </div>
                    <div class="lnkbuttonGroup" v-show="rightBoxType == 'modify'" style="margin-left:20px;">
                        <el-button  type="primary"  @click="modifyDeviceSubmit" :loading="showSubmitLoading"><%=rb.getString("QueDing")%></el-button>
                        <el-button  @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
                    </div>
                    <div class="lnkbuttonGroup" v-show="rightBoxType == 'addGroup'" style="margin-left:20px;">
                        <el-button  type="primary"  @click="addGroupSubmit" :loading="showSubmitLoading"><%=rb.getString("QueDing")%></el-button>
                        <el-button  @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
                    </div>
                </div>
            </div>
        </div>
    </div>
    <!--移动设备到设备组-->
    <el-dialog :title='moveDevice.dialogTitle' :visible.sync="moveDevice.showMoveInfo" :width="moveDevice.windowWidth">
        <el-ctable ref="ctableGroup" :url="moveDevice.groupUrl" :height='moveDevice.height'>
            <el-table-column width="30">
                <template slot-scope="scope">
                    <el-radio v-model="moveDevice.groupId" :label="scope.row.id">&nbsp;</el-radio>
                </template>
            </el-table-column>
            <el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="150" prop="group_name"></el-table-column>
        </el-ctable>
        <div>
            <el-button type="primary" @click='moveTrue'><%=rb.getString("QueDing")%></el-button>
            <el-button @click="moveDevice.showMoveInfo = false"><%=rb.getString("QuXiao")%></el-button>
        </div>
    </el-dialog>
    <!--导入结果弹窗-->
    <el-dialog title='<%=rb.getString("JieGuo")%>' :visible.sync="importResultDialog" width="400" class='importResultWarp'>
		<div v-if="isCSVFile">
			<div class='importResultDivs commonFlex'>
				<p class='commonSize14'>{{csvMsgTips}}</p>
			</div>
		</div>
		<div v-else>
			<p class='commonText14 importResultDivs'><%=rb.getString("SheBeiDaoRuJieGuo")%></p>
			<div class='importResultDivs commonFlex'>
				<p class='commonSize14'><%=rb.getString("DaoRuChengGongShuLiang")%></p>
				<p class='exportTemplateIcon' style='color: #67D972;'>{{checkSuccessCount}}</p>
			</div>
			<div class='importResultDivs commonFlex'>
				<p class='commonSize14'><%=rb.getString("DaoRuShiBaiShuLiang")%></p>
				<p class='exportTemplateIcon' style='color: #E88282;'>{{checkUnSuccessCount}}</p>
			</div>
			<el-button type="primary" @click='downloadFailClick' class='downloadBtnTips'><%=rb.getString("XiaZaiShiBaiMingXi")%></el-button>
		</div>
    </el-dialog>
    <!-- 上传文件的用的表单 -->
    <form enctype="multipart/form-data" method="post" id="importDeviceForm">
        <input name="fileSize"  value="" hidden="true">
        <input name="operType" value="" hidden="true">
        <input name="uploadFile"  id="importDeviceFile"  type="file" style="display: none;">
    </form>
    <!-- 下载模板用的表单 -->
    <form id="exportDeviceForm" style="display:none" method="post"></form>
    <!--告警邮件结果，告警详情，告警图表  slide -->
    <el-slide  ref="sharingSlide" :url='sharingSlideUrl' :title="sharingSlideTitle" :footer="sharingSlideFooter" :header="sharingSlideHeader" :position="sharingSlidePosition"
        :height="sharingSlideHeight" :modal='modal' :width='sharingSlideWidth'  @cancel="sharingSlideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
    </el-slide>

    <!-- add Site -->
    <el-dialog ref="siteAdd" style="top: 100px;"
        :modal="false"
        :close-on-click-modal="false"
        :append-to-body="false"
        title='New Site'
        :width="400"
        :visible.sync="siteAddShow"
    >
        <el-form ref="addSiteForm" :model="addSiteForm" :rules="addSiteRule" label-position="top" size="mini">
            <el-form-item label='<%=rb.getString("ZhanZhiMingCheng")%>' style="margin-bottom: 20px;" prop="siteName">
                <el-input v-model="addSiteForm.siteName" maxlength="64"></el-input>
            </el-form-item>
            <el-form-item label='<%=rb.getString("WeiDu")%>' style="margin-bottom: 20px;" prop="latitude">
                <el-input v-model="addSiteForm.latitude"></el-input>
            </el-form-item>
            <el-form-item label='<%=rb.getString("JingDu")%>' prop="longitude">
                <el-input v-model="addSiteForm.longitude"></el-input>
            </el-form-item>
        </el-form>

        <div slot="footer" style="padding: 0 10px;">
            <el-button type="primary" @click="addSiteSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="siteAddShow = false"><%=rb.getString("QuXiao")%></el-button>
        </div>
    </el-dialog>

	<el-dialog title='<%=rb.getString("QueRen")%>' top="30vh" width="550"
		:visible.sync="locationReasonableShow" 
		:modal="true"
		:close-on-click-modal="false"
	>
		<div>
			<div>{{locationTips}}</div>
			<span>
				<%=rb.getString("ChongSheJiChuWeiZhi1")%> 
				[
					{{currentRow.latitude}}, {{currentRow.longitude}}
				] 
				<%=rb.getString("ChongSheJiChuWeiZhi2")%></span>
		</div>
		<div slot="footer" style="text-align: right;">
			<el-button type="primary" @click="activeWithLocation"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="locationReasonableShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>

<script>
if(window.egwRegisterVue) {
	try {
		window.egwRegisterVue.$destroy();
	}catch(e){}
}
window.egwRegisterVue = new Vue({
	el:'#egwRegister',
	data(){
		var vm = this,
			lessThenLength = function(str,num){
				str += '';
				var index = str.indexOf('.');
				if(index > -1){
					return str.substring(index+1).length <= num
				}
				return true;
			};

		var validatorDeviceIp = (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				
				if(value == ""){
					callback(new Error('<%=rb.getString("IPDiZhiFeiFa")%>'))
				}else if(reg.test(value)){
					callback()
				}else{
					callback(new Error('<%=rb.getString("IPDiZhiFeiFa")%>'))
				}
			},
			
			validatorDevicePort = (rule,value,callback) => {
				
				if(value === ''){
					callback(new Error('<%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> 0~65535'))
				}else if(vm.isNumeric(value)&& parseInt(value)>=0 && parseInt(value)<=65535){
					callback();
				}else{
					callback(new Error('<%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> 0~65535'))
				}
			},
			validateGroupName = (rule,value,callback) => {
				var reg = /^[a-zA-Z0-9_\u4e00-\u9fa5,\s]{1,50}$/;

				if(this.addGroupType == 'addOneGroup'){
					if(value){
						if(reg.test(value)){
							callback()
						}else{
							callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
						}
					}else{
						callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
					}
				}else{
					callback()
				}
				
			},
			validateSubGroupName = (rule,value,callback) => {
				var reg = /^[a-zA-Z0-9_\u4e00-\u9fa5,\s]{1,50}$/;

				if(this.addGroupForm.addSubCheck){
					if(value){
						if(reg.test(value)){
							callback()
						}else{
							callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
						}
					}else{
						callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
					}
				}else{
					callback()
				}
				
			},
			validateDevice = function(rule,value,callback) { // 校验设备
				if(value.length == 0) {
					// callback('<%=rb.getString("QingXuanZeSheBei")%>');
					callback();
				}else {
					callback();
				}
			},
			validatorNum = (rule,value,callback) => {
				var serialNumber = value||'',
					list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;}),
					temp = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/,
					noColTemp = /^([A-Fa-f0-9]{2}){6}$/,
					tempSn = /^(\d|[a-zA-Z]|-|\s){1,30}$/;
					 
				if(this.addDeviceForm.deviceType == 'WCG'){
					tempSn = /^(\d|[a-zA-Z]|-|\s){1,32}$/;
				}else if(this.addDeviceForm.deviceType == 'gNB'){
					tempSn = /^(\d|[a-zA-Z]|-|\s){1,45}$/;
				
				}
				if(this.isDeviceMoreParams == 'true' && this.addDeviceForm.deviceType == 'eNB'){
					if(serialNumber && tempSn.test(serialNumber)){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
				}else{
					if(serialNumber != null && serialNumber.length != 0){
						var nameFlag;
							if(this.addDeviceForm.inputType == 'mac'){
								nameFlag = list.every(function(item,index){
									return (temp.test(item) ||  noColTemp.test(item)) 
								})
							}else{
								nameFlag = list.every(function(item,index){
									return tempSn.test(item)
								})
							}
						if(nameFlag){
							callback()
						}else{
							if(this.addDeviceForm.inputType == 'mac'){
								callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
							}else{
								callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
							}
						}
					}else if (serialNumber == null || serialNumber.length == 0) {
						if(this.addDeviceForm.inputType == 'mac'){
							callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
						}else{
							callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
						}
					}else{
						callback();
					}
				}
			},
			validatorRequired = function(rule,value,callback) {
				
				if(vm.isDeviceMoreParams == 'true' && vm.addDeviceForm.deviceType == 'eNB' && vm.showOrHideCol !=="true"){
					if(value){
						callback();
					}else {
						callback('<%=rb.getString("QingShuRuBiTianXiang")%>');
					}
				}else {
					callback();
				}
			},
			validatorIP = function(rule,value,callback) {
				if( !isIPv4(value)) {
					callback('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>');
				}else {
					callback();
				}
			},
			validTAC = function(rule,value,cb) {
				var reg = /^(0|[1-9][0-9]*)$/,
					val = value;

				if (!reg.test(val) || (val < 0 || val > 65535)) {
					cb('Integer, 0-65535');
				}else{
					cb();
				}
			},
			// 经度 表单验证规则
			longitudeValidator = function(rule,value,cb) {
				if(value) {
					if(!isNaN(value) && Math.abs(value)<=180 && lessThenLength(Math.abs(value),6)) {
						cb();
					}else {
						cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
					}
				}else {
					cb();
				}
			},
			// 维度 表单验证规则
			latitudeValidator = function(rule,value,cb) {
				if(value) {
					if(!isNaN(value) && Math.abs(value)<=90 && lessThenLength(Math.abs(value),6)) {
						cb();
					}else {
						cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
					}
				}else {
					cb();
				}
			},
			// 高度 表单验证规则
			heightValidator = function(rule,value,cb) {
				if(value) {
					if(value<=99999999 && value-0>=0) {
						cb();
					}else {
						cb('<%=rb.getString("QuZhiFanWei")%>: 0-99999999');
					}
				}else {
					cb();
				}
			},
			// 距离 表单验证规则
			distanceValidator = function(rule,value,cb) {
				if(value) {
					if(value<=99999999 && value-0>=0) {
						cb();
					}else {
						cb('<%=rb.getString("QuZhiFanWei")%>: 0-99999999');
					}
				}else {
					cb();
				}
			},
			latValid = function(rule,value,cb){
				value = value + '';

				if(value) {
					if(isNaN(value)) {
						cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
					}else if(value<-90 || value>90){
						cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
					}else {
						var arr = value.split('.'),
							precision = arr[1]||'';
						if(precision.length>6) {
							cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
						}else cb();
					}
				}else {
					if(rule.required == true) {
						cb('Required');
					}else {
						cb();
					}
				}
			},
			/**
			* 经度校验
			* @param rule{object}：校验配置的规则
			* @param value{string}: 经度值
			* @param cb{function}：回调方法
			**/
			lonValid = function(rule,value,cb){
				value = value + '';

				if(value) {
					if(isNaN(value)) {
						cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
					}else if(value<-180 || value>180){
						cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
					}else {
						var arr = value.split('.'),
							precision = arr[1]||'';
						if(precision.length>6) {
							cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
						}else cb();
					}
				}else {
					if(rule.required == true) {
						cb('Required');
					}else {
						cb();
					}
				}
			},
            validDeviceRuleTAC = function(rule,value,callback) {
                var message = vm.deviceType == 'eNB' ? 'eg: 1,2,3,1-3;Range, 0-65535' : 'eg: 1,2,3,1-3;Range, 0-16777215';
                if(vm.addGroupForm.enable == '1' && (vm.addGroupForm.matching_mode == 'tac' || vm.addGroupForm.matching_mode == 'lac')){
                    if(value == '' || value == null || value == undefined){
                        callback(message);
                    }else{
                        if(vm.deviceType == 'eNB'){
                            if(!vm.isValidIntegerTacRange(value,0,65535)){
                                callback(message);
                            }else{
                                callback();
                            }
                        }else{
                            if(!vm.isValidIntegerTacRange(value,0,16777215)){
                                callback(message);
                            }else{
                                callback();
                            }
                        }
                    }
                }else{
                    callback();
                }
            },
            validNameContainsContent = function(rule,value,callback) {

                if(vm.addGroupForm.enable == '1' && vm.addGroupForm.matching_mode == 'deviceName'){
                    var hasError = false;
                    
                    // 遍历所有输入框，检查是否有空值
                    vm.nameContainsContentList.forEach(function(filter, index){
                        var isEmpty = !filter.value || filter.value.trim() === '';
                        if(isEmpty){
                            vm.$set(filter, 'hasError', true);
                            hasError = true;
                        } else {
                            vm.$set(filter, 'hasError', false);
                        }
                    });
                    
                    if(hasError){
                        callback(new Error('Please input required field'));
                        return;
                    }
                    
                    callback();
                }else{
                    // 清除所有错误状态
                    vm.nameContainsContentList.forEach(function(filter){
                        vm.$set(filter, 'hasError', false);
                    });
                    callback();
                }
            },
			validMechanicalDowntilt = function(rule,value,callback) {
				var msg = 'Range: 0-9';
				
				if(value === '') {
					callback();
				}else {
					if(isNaN(value) || value - 0 < 0 || value - 9 > 0) {
						callback(msg)
					}else {
						callback();
					}
				}
			},
			validVertical3dB = function(rule,value,callback) {
				var msg = 'Range: 1-9';
				
				if(value === '') {
					callback();
				}else {
					if(isNaN(value) || value - 1 < 0 || value - 9 > 0) {
						callback(msg)
					}else {
						callback();
					}
				}
			},
			validHorizontalAzim = function(rule,value,callback) {
				var msg = 'Range: 0-359';
				
				if(value === '') {
					callback();
				}else {
					if(isNaN(value) || value - 0 < 0 || value - 359 > 0) {
						callback(msg)
					}else {
						callback();
					}
				}
			};

		return {
			locationTips: '',
			locationReasonableShow: false,
			locationParams: {
				op_state: '',
				ids: ''
			},
			currentRow: {},

			detectable: locationDetection == true,
			addDeviceForm:{
				serialNumber:'',
				group_id:'',
				selectType:'0',
				inputType:'sn',
				deviceType:'eNB',

				manufacturer: '',
				module_type: '',
				host_name: '',
				province: '',
				city: '',
				district: '',
				township: '',
				sub_grid: '',
				sub_branches: '',
				sub_station_code: '',
				sub_station_name: '',
				sub_distribute_system: '',
				sub_cell_id: '',
				sub_cell_name: '',
				sub_talist: '',
				over_scen_attr: '',
				maxtxpower: '',
				device_access_mode: '',
				rx_port_number: '',
				tx_port_number: '',
				omc_ip: '',
				cell_ip: '',
				tac: '',
				software_version: '',
				net_grade: '',
				gps_longitude:'',
				gps_latitude:'',
				install_address: '',
				access_net_date: '',
				agent_maintain: '',
				contact_person: '',
				contact_number: '',
				device_status:1,
				uplink_broadband_account:'',
				site_id: '',
				circuit_ref: '',
				circuit_jo: '',
				service_status: '',
				rom: '',

				mechanical_downtilt: '',
				vertical_3dB_beam_width: '',
				horizontal_azimuth: ''
			},
			addDeviceFormRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'blur'}
				],
				manufacturer:[{validator:validatorRequired}],
				module_type:[{validator:validatorRequired}],
				host_name:[{validator:validatorRequired}],
				province:[{validator:validatorRequired}],
				city:[{validator:validatorRequired}],
				district:[{validator:validatorRequired}],
				township:[{validator:validatorRequired}],
				sub_cell_id:[{validator:validatorRequired}],
				sub_cell_name:[{validator:validatorRequired}],
				over_scen_attr:[{validator:validatorRequired}],
				omc_ip:[{validator:validatorIP}],
				cell_ip:[{validator:validatorIP}],
				tac:[{validator:validTAC}],
				install_address:[{validator:validatorRequired}],
				contact_person:[{validator:validatorRequired}],
				contact_number:[{validator:validatorRequired}],
				uplink_broadband_account:[{validator:validatorRequired}],

				mechanical_downtilt: [{validator: validMechanicalDowntilt}],
				vertical_3dB_beam_width: [{validator: validVertical3dB}],
				horizontal_azimuth: [{validator: validHorizontalAzim}]
			},
			modifyDeviceForm:{
				serial_number:'',
				macaddress:'',
				longitude: '',
				latitude: '',
				height: '',
				distance: '',
				operator_code: operator_code,
				isCheckInActive: '',

				manufacturer: '',
				module_type: '',
				host_name: '',
				province: '',
				city: '',
				district: '',
				township: '',
				sub_grid: '',
				sub_branches: '',
				sub_station_code: '',
				sub_station_name: '',
				sub_distribute_system: '',
				sub_cell_id: '',
				sub_cell_name: '',
				sub_talist: '',
				over_scen_attr: '',
				maxtxpower: '',
				device_access_mode: '',
				rx_port_number: '',
				tx_port_number: '',
				omc_ip: '',
				cell_ip: '',
				tac: '',
				software_version: '',
				net_grade: '',
				install_address: '',
				access_net_date: '',
				agent_maintain: '',
				contact_person: '',
				contact_number: '',
				device_status:1,
				uplink_broadband_account:'',
				site_id: '',
				circuit_ref: '',
				circuit_jo: '',
				service_status: '',
				rom: '',

				egwName:'',
				egwSn:'',
				egwDescription:'',
				hardwareModel:'',

				mechanical_downtilt: '',
				vertical_3dB_beam_width: '',
				horizontal_azimuth: '',

                remark: '',
			},
			isMoreParams:{
				manufacturer: '',
				module_type: '',
				host_name: '',
				province: '',
				city: '',
				district: '',
				township: '',
				sub_grid: '',
				sub_branches: '',
				sub_station_code: '',
				sub_station_name: '',
				sub_distribute_system: '',
				sub_cell_id: '',
				sub_cell_name: '',
				sub_talist: '',
				over_scen_attr: '',
				maxtxpower: '',
				device_access_mode: '',
				rx_port_number: '',
				tx_port_number: '',
				omc_ip: '',
				cell_ip: '',
				tac: '',
				software_version: '',
				net_grade: '',
				install_address: '',
				access_net_date: '',
				agent_maintain: '',
				contact_person: '',
				contact_number: '',
				device_status:1,
				uplink_broadband_account:'',
			},
			isThailandTrueParams:{
				host_name: '',
				sub_station_name:'',
				install_address: '',
				contact_number: '',
				site_id: '',
				circuit_ref: '',
				circuit_jo: '',
				service_status: '',
				rom: '',

			},
			defaultAddDeviceForm:{},
			defaultModifyDeviceForm:{},
			isFormDefaultsInitialized: false,
			modifyDeviceFormRules:{
				longitude: [
					{validator: longitudeValidator}
				],
				latitude: [
					{validator: latitudeValidator}
				],
				height: [
					{validator: heightValidator}
				],
				distance: [
					{validator: distanceValidator}
				],
				manufacturer:[{validator:validatorRequired}],
				module_type:[{validator:validatorRequired}],
				host_name:[{validator:validatorRequired}],
				province:[{validator:validatorRequired}],
				city:[{validator:validatorRequired}],
				district:[{validator:validatorRequired}],
				township:[{validator:validatorRequired}],
				sub_cell_id:[{validator:validatorRequired}],
				sub_cell_name:[{validator:validatorRequired}],
				over_scen_attr:[{validator:validatorRequired}],
				omc_ip:[{validator:validatorIP}],
				cell_ip:[{validator:validatorIP}],
				tac:[{validator:validTAC}],
				install_address:[{validator:validatorRequired}],
				contact_person:[{validator:validatorRequired}],
				contact_number:[{validator:validatorRequired}],
				uplink_broadband_account:[{validator:validatorRequired}],

				mechanical_downtilt: [{validator: validMechanicalDowntilt}],
				vertical_3dB_beam_width: [{validator: validVertical3dB}],
				horizontal_azimuth: [{validator: validHorizontalAzim}]
			},
			moveDevice:{ // 移动设备对象
				groupUrl:'', //tabUrl
				showMoveInfo:false, // 移动设备弹窗
				height:'400px', // 表格高度
				dialogTitle:'', //弹窗title
				windowWidth:'', // 弹窗宽度
				groupId:'', // 设备组ID
				code:'', //设备组code值
			},
			height:'100%',
			menusGroup:[],
			deviceUrl:'',
			params_device:{
				group_id:'',
				search_text:'',
				timeZone:timeZone,
			},
			dialogTitle:'',
			groupOptType:'',
			rowDataOneGroup:[],
			rowDataGroup:[],
			rowDataDevice:[],

			addGroupForm:{
				topGroupName:'',
				addSubCheck:false,
				subGroupName:'',
                enable: '0',
                matching_mode: 'deviceName',
                name_contains: '',
                tac_rag: '',
				desc:'',
				deviceType:[]
			},
			addGroupFormRules:{
				topGroupName:[
					{validator:validateGroupName,trigger:'blur'}
				],
				subGroupName:[
					{validator:validateSubGroupName,trigger:'blur'}
				],
                name_contains:[{validator:validNameContainsContent}],
                tac_rag:[{validator:validDeviceRuleTAC}],
			},
			addGroupDeviceActiveName:'',
			addGroupDeviceList:[
				{
					label:'eNB',
					refStr:'addGroupEnbTable',
					isShow:false,
					leftUrl:'${ctx}/cell/cpeinfos/getEnbList.action',
					oldRightUrl:'${ctx}/system/device/enodeb/queryENBInfoPageList.action?group_id=',
					rightUrl:'',
					title:['','<%=rb.getString("YiXuan")%>'],
					rowKey:'serial_number',
					messages:{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'},
					search_text:'',
					queryParams: {
						search_text: '',
						group_id:'',
						timeZone: timeZone
					},
				},
				{
					label:'gNB',
					refStr:'addGroupGnbTable',
					isShow:false,
					leftUrl:'${ctx}/cell/cpeinfos/getEnbList.action',
					oldRightUrl:'${ctx}/system/device/enodeb/queryENBInfoPageList.action?isGnb=1&group_id=',
					rightUrl:'',
					title:['','<%=rb.getString("YiXuan")%>'],
					rowKey:'serial_number',
					messages:{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'},
					search_text:'',
					queryParams: {
						search_text: '',
						group_id:'',
						isGnb:1,
						timeZone: timeZone
					},
				},
				{
					label:'CPE',
					refStr:'addGroupCpeTable',
					isShow:false,
					leftUrl:'${ctx}/system/device/cpe/queryCPEInfoPageList.action',
					oldRightUrl:'${ctx}/system/device/cpe/queryCPEInfoPageList.action?group_id=',
					rightUrl:'',
					title:['','<%=rb.getString("YiXuan")%>'],
					rowKey:'cpe_code',
					messages:{placeholder:'<%=rb.getString("CPEXuLieHao")%>/<%=rb.getString("MACDiZhi")%>'},
					search_text:'',
					queryParams: {
						search_text: '',
						group_id:'',
						timeZone: timeZone,
						like_fields: 'serial_number,macaddress'
					},
				},
				{
					label:'WCG',
					refStr:'addGroupEgwTable',
					isShow:false,
					leftUrl:'${ctx}/egw/register/getEGWListByGroupId.action',
					oldRightUrl:'${ctx}/egw/register/getEGWListByGroupId.action?group_id=',
					rightUrl:'',
					title:['','<%=rb.getString("YiXuan")%>'],
					rowKey:'egwCode',
					messages:{placeholder:'<%=rb.getString("eGWBianMa")%>'},
					search_text:'',
					queryParams: {
						search_text: '',
						timeZone: timeZone
					},
				},
			],

			deviceGroupOptions:[],
			deviceSelection:[],
			groupData:[],
			queryGroupSearchText:'',
			defaultexpandedKeys:[],
			defaultCheckedKeys:[],

			deviceType:'',
			bulkSelectShow:false,
			search_text:'',
			placeholderText:'<%=rb.getString("QingShuRu")%>',
			deviceTypeList:[],
			deviceListRightBoxShow:false,
			rightBoxTitle:'',
			rightBoxType:'',

			addSelectFlag:false,        //标识是否选择了文件
			typeFlag:true,           //校验已选择的文件格式
			addFileName:'',
			addFileParams:{},            //上传文件时自定义的参数   
			addUploadFileURL: '',
			isDeviceMoreParams:"${manageConfigFlag}",
			showOrHideCol : "${enbAdditionalColShow}",
			exportDeviceType:[],
			deviceExportErrorShow:false,

			addGroupOptions:[],
			addGroupType:'',
			tabUrlList:{
				'eNB':'${ctx}/cell/cpeinfos/getEnbList.action',
				'gNB':'${ctx}/cell/cpeinfos/getEnbList.action?isGnb=1',
				'CPE':'${ctx}/system/device/cpe/queryCPEInfoPageList.action',
				'WCG':'${ctx}/egw/register/getEGWListByGroupId.action?type=4',
			},
			valCodes:{
				'eNB':'small_cell_code',
				'gNB':'small_cell_code',
				'CPE':'cpe_code',
				'WCG':'egwCode'
			},
			subCodes:{
				'eNB':'ids',
				'gNB':'ids',
				'CPE':'cpeCodes',
				'WCG':'egwCodes'
			},

			importResultDialog: false,
			checkSuccessCount: '0',
			checkUnSuccessCount: '0',
			batchOperation:batchOperation,
			isCSVFile: false,
			csvMsgTips: '',

			sharingSlideUrl:'',
			sharingSlideTitle:'',
			sharingSlideFooter:'',
			sharingSlideHeader:'',
			sharingSlidePosition:'',
			sharingSlideHeight:'',
			sharingSlideWidth:'',

			siteNames: [],
			siteEnable: supportTopoSite,
			siteAddShow: false,
			addSiteForm: {
				siteName: '',
				latitude: '',
				longitude: '',
				cellCodes: ''
			},
			addSiteRule: {
				siteName: [{required: true, message: 'Required'}],
				latitude: [{validator: latValid, required: true}],
				longitude: [{validator: lonValid, required: true}],
			},
            deviceRuleOrder:'',
            deviceRuleId:'',
            isSupportGSM: supportGSM,
            showSubmitLoading:false,
            showSiteId: registerSiteIdShow,

            // 过滤器选项配置 
			filterConditionOptions: [
                {label: '<%=rb.getString("BaoHan")%>', value: 'contain', minIndex: 0},
				{label: '<%=rb.getString("BuBaoHan")%>', value: 'notContain', minIndex: 0},
				{label: '<%=rb.getString("YiShenMoKaiShi")%>', value: 'startWith', minIndex: 0},
				{label: '<%=rb.getString("YiShenMoJieShu")%>', value: 'endWith', minIndex: 0}
			],
            // 或 与 下拉选项配置
            conditionAndOrOptions: [
                {label: '<%=rb.getString("Yu")%>', value: 'and'},
                {label: '<%=rb.getString("Huo")%>', value: 'or'}
			],
			// 过滤器列表
			nameContainsContentList: [
				{
					condition: 'contain',
					value: '',
					hasError: false
				}
			],
			// remark列label自定义
			editingRemarkLabel: false,
			remarkLabelInput: 'Remark',
			currentRemarkLabel: 'Remark',
		}
	},
	methods:{
		// 初始化
		init(){	
			var vm = this, 
				deviceTypeList = [],
                netTypeCodes = {
					'enb':'eNB',
					'gnb':'gNB',
					'cpe':'CPE',
					'egw':'WCG'
				},
				codes = [
					{label:'eNB',icon:'el-icon el-icon-menu-eNB',code:'CODE_ENB_DEVICE_REGISTER'},
					{label:'gNB',icon:'el-icon el-icon-menu-gNB',code:'CODE_GNB_DEVICE_REGISTER'},
					{label:'CPE',icon:'el-icon el-icon-menu-CPE',code:'CODE_CPE_DEVICE'},
					{label:'WCG',icon:'el-icon el-icon-menu-eGW',code:'CODE_EGW'},
				];
			
			var tabType = sysMain.$refs.nav.editableTabs.filter((item)=>{
                return ['400001','400002','400003','400004'].includes(item.id);
            })[0].netType;
			vm.deviceType = netTypeCodes[tabType];
			codes.map((item)=>{
				if(item.label == tabType){
					deviceTypeList.push(item)
				}
			})
			vm.deviceTypeList = deviceTypeList;
			vm.queryGroupList(vm.queryGroupSearchText);
			vm.querySiteNames();
            vm.getCustomLabelData();
			if(!vm.isFormDefaultsInitialized){
                vm.defaultModifyDeviceForm = JSON.parse(JSON.stringify(vm.modifyDeviceForm));
                vm.defaultAddDeviceForm = JSON.parse(JSON.stringify(vm.addDeviceForm));
                vm.isFormDefaultsInitialized = true;
			}
			if(vm.$refs.sharingSlide) vm.$refs.sharingSlide.hide();
		},
		querySiteNames() {
			var vm = this,
				url = '${ctx}/site/getSiteInfosList.action',
				params = {
					qryFields: 'siteName'
				};

			axios.post(url, stringify(params)).then(function(res) {
				vm.siteNames = res.data || [];
			});
		},
		// 设备组查询
		queryGroupList(val){
			var vm =this,
				typeCode={
					'eNB': 0,
					'gNB': 1,
					'CPE': 2,
					'WCG': 4
				},
				params={
					search_text:val,
					type: typeCode[vm.deviceType]
				};
			vm.queryGroupSearchText = val;
			vm.defaultexpandedKeys =[];
			vm.defaultCheckedKeys = [];
			vm.deviceUrl = '';
			axios.post('${ctx}/system/deviceGroup/getFullDeviceGroupList.action',stringify(params)).then(function(response){
				let data = response.data
				if(data.rows.length>0){
					data.rows.map((item)=>{
						item.isEdit = 'false';
					})
					vm.groupData = data.rows;
					vm.defaultexpandedKeys.push(vm.groupData[0].id);
					vm.defaultCheckedKeys.push(vm.groupData[0].children[0].id);
					vm.params_device.group_id =vm.groupData[0].children[0].id;
					vm.rowDataGroup = vm.groupData[0].children[0];
					vm.$nextTick(function(){
						vm.$refs.groupTree.setCurrentKey(vm.params_device.group_id);
						vm.deviceUrl = vm.tabUrlList[vm.deviceType];
					})
				}
			}).catch(function(error){})
		},
		queryDeviceGroupOption(){
			var vm = this;
			axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then(function(response){
				let data = response.data
				vm.deviceGroupOptions = data;
				if(vm.deviceGroupOptions.length>0){
					vm.addDeviceForm.group_id = vm.deviceGroupOptions[0].id;
				}
			}).catch(function(error){});
		},
		// 添加设备
		addDevice(){
	    	var vm = this;

			
			vm.rightBoxTitle = 'Add';
			vm.addGroupType = '';
			vm.rightBoxType = 'add';
			vm.addDeviceForm.deviceType = vm.deviceType;
			vm.queryDeviceGroupOption();
			Object.keys(vm.addDeviceForm).forEach(function(key){
				if(key != 'group_id' && key != 'deviceType'){
					vm.addDeviceForm[key] = vm.defaultAddDeviceForm[key]
				}
			});
			vm.deviceListRightBoxShow = true;
            vm.showSubmitLoading = false;
		},
		// 添加一级 设备组
		addDeviceGroup(){
			var vm = this;

			vm.rightBoxTitle = 'Add Device Group';
			vm.rightBoxType = 'addGroup';
			vm.addGroupType = 'addOneGroup';
			vm.deviceListRightBoxShow = true;
            vm.showSubmitLoading = false;
			vm.clearAddDeviceGroupTableSelect();
			vm.queryAddGroupOption();
			vm.$nextTick(()=>{
				vm.$refs.addGroupForm.clearValidate();
			})
			
		},
		// 初始化设备组弹窗 表格已选
		clearAddDeviceGroupTableSelect(){
			var vm = this;
			vm.addGroupDeviceList.map((item)=>{
				if(item.label == vm.deviceType){
					item.isShow = true;
				}else{
					item.isShow = false;
				}
				item.search_text = '';
				item.queryParams.search_text = item.search_text;
				item.rightUrl = '';
			})

			vm.addGroupForm.deviceType.push(vm.deviceType);
			vm.addGroupDeviceActiveName = vm.deviceType;
		},
		//请求设备组下拉
		queryAddGroupOption(){
			var vm = this;
			axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action').then(function(response){
				let data = response.data
				vm.addGroupOptions = data;
			}).catch(function(error){});
		},
		//点击页面其他地方菜单收起
		handerClose(){ 
	        this.$refs.menuGroup.hide();
	    },
		groupRowClick(data,node,ev){
			var vm = this;
			if(!data.children){
				vm.rowDataGroup = data;
				vm.params_device.group_id = data.id;
				vm.$nextTick(function(){
					vm.deviceUrl = vm.tabUrlList[vm.deviceType];
					vm.$refs.ctableDevice.clearSelection();
				})
			}else{
				vm.$nextTick(function(){
					vm.deviceUrl = vm.tabUrlList[vm.deviceType];
					vm.$refs.ctableDevice.clearSelection();
				})
			}
		},
		/**
		 * 点击设备组操作 生成下拉选项
		 * @param row:当前点击项数据 
		 * 操作项： 1.信息  2.修改  3.删除  
		*/
		groupOpClick(node,data,ev){
			var vm = this,isSubGroup = false,showEditFlag = true,showDelFlag = true,editDisFlag=false;
			
			if(data.children){
				isSubGroup = false;
				vm.rowDataOneGroup = data;
				vm.groupOptType = 'oneGroup';
				data.children.map((item)=>{
					if(item.write !== '1'){
						showDelFlag = false
					}
				})
			}else{
				vm.groupOptType = 'subGroup';
				vm.rowDataGroup = data;
				isSubGroup = true;
				if(data.write !== "1" ){
					showEditFlag = false;
					showDelFlag = false;
				}
			}
			if(data.built_in == "1" ){
				editDisFlag = true;
			}
	    	vm.menusGroup= [
		        {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info',show:isSubGroup},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'edit',show:showEditFlag,disable:editDisFlag},
		        {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del',show:showDelFlag,disable:editDisFlag}
		    ]
	    	
	    	vm.$nextTick(function(){
	    		document.body.click();
				vm.$refs.menuGroup.show(ev);
	    	});
			event.stopPropagation();
		},
		
		/**
		 * 设备组更多操作栏 单点方法 
		 * @param ev:当前点项
		*/
	    clickMenu(ev){ 
	    	var vm = this;
	    	var codes = {
	    		info:vm.viewSubGroupInfo,
	    		edit:vm.modifyGroup,
	    		del:vm.deleleGroup
	    	}
	    	if(codes[ev.code]){
				if(vm.groupOptType == 'oneGroup'){
					codes[ev.code](vm.rowDataOneGroup.id,vm.rowDataOneGroup)
				}else{
					codes[ev.code](vm.rowDataGroup.id,vm.rowDataGroup)
				}
	    		
	    	}
	    },
		// 添加二级 设备组
		addSubGroup(node,row,ev){
			var vm = this;

			vm.rightBoxTitle = 'Add Subgroup';
			vm.rightBoxType = 'addGroup';
			vm.addGroupType = 'addSubGroup';
			vm.rowDataOneGroup = row;
			vm.addGroupForm.addSubCheck = true;
			vm.clearAddDeviceGroupTableSelect();
			vm.queryAddGroupOption();
			vm.deviceListRightBoxShow = true;
            vm.showSubmitLoading = false;
			vm.$nextTick(()=>{
				vm.$refs.addGroupForm.clearValidate();
			})
		},
		/**
		* 查看设备组详情
		* @param id 传入当前数据的id	
		*/
	    viewSubGroupInfo(id,data){
			var vm = this,
				params={
					desc:data.description,
					subGroupName:data.group_name,
					deviceType:[]
				},
				randomCode = Math.random().toString();
			
			vm.rightBoxTitle = 'View Subgroup';
			vm.rightBoxType = 'addGroup';
			vm.addGroupType = 'viewSubGroup';
			vm.addGroupForm.addSubCheck = true;
			params.deviceType.push(vm.deviceType);
			vm.deviceListRightBoxShow = true;
			vm.showSubmitLoading = false;
			vm.$nextTick(()=>{
				vm.$refs.addGroupForm.resetFields();
				vm.addGroupDeviceList.map((item)=>{
					item.isShow = params.deviceType.includes(item.label) ? true : false;
					item.search_text = '';
					item.queryParams.search_text = item.search_text;
					item.rightUrl = item.oldRightUrl + id + '&randomCode=' + randomCode;
				})
				Object.assign(vm.addGroupForm,params);
				if(params.deviceType.length>0){
					vm.addGroupDeviceActiveName = params.deviceType[0];
				}
                vm.getDeviceGroupRuleInfo();
			})
			vm.queryAddGroupOption();
	    },
		/**
		* 修改设备组
		* @param id 传入当前数据的id	
		*/
	    modifyGroup(id,data){
			var vm = this,
				params={
					topGroupName:vm.rowDataGroup.group_name,
					desc:vm.rowDataGroup.description,
					deviceType:[]
				},
				randomCode = Math.random().toString();
			if(data.children){
				vm.groupData.map((item)=>{
					if(item.id == id){
						item.isEdit = 'true';
					}
				})
			}else{
				params.subGroupName = vm.rowDataGroup.group_name
				vm.rightBoxTitle = 'Modify Subgroup';
				vm.rightBoxType = 'addGroup';
				vm.addGroupType = 'editSubGroup';
				vm.addGroupForm.addSubCheck = true;
				params.deviceType.push(vm.deviceType);
				vm.deviceListRightBoxShow = true;
                vm.showSubmitLoading = false;
				vm.$nextTick(()=>{
					vm.$refs.addGroupForm.resetFields();
					vm.addGroupDeviceList.map((item)=>{
						item.isShow = params.deviceType.includes(item.label) ? true : false;
						item.search_text = '';
						item.queryParams.search_text = item.search_text;
						item.rightUrl = item.oldRightUrl + id + '&randomCode=' + randomCode;
					})
					Object.assign(vm.addGroupForm,params);
					if(params.deviceType.length>0){
						vm.addGroupDeviceActiveName = params.deviceType[0];
					}
                    vm.getDeviceGroupRuleInfo();
				})
				vm.queryAddGroupOption();
			}
	    },
		/**
		* 删除设备组
		* @param id 传入当前数据的id	
		*/
	    deleleGroup(id,data){
	    	var vm = this,
				delTips = '',
				urls = '',
				params = {};
			if(data.children){
				delTips = '<%=rb.getString("ZuNeiZiJiSheBeiZuJiSheBeiHuiBeiZiDongYiZhi")%>';
				params.groupId = id;
				urls = '${ctx}/system/deviceGroup/delTopDevice.action'
			}else{
				delTips = '<%=rb.getString("ZuNeiSheBeiHuiBeiZiDongYiDongDaoMoRenFenZu")%>';
				params.id = id;
				urls = "${ctx}/system/deviceGroup/deleteDeviceGroup.action"
			}
			var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueDingShanChuSheBeiZu")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ delTips +'</div>';
			vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
				customClass:'',
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				dangerouslyUseHTMLString:true
			}).then(()=>{
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});	
						vm.queryGroupList(vm.queryGroupSearchText);
					}else{
						vm.$message.error(data.message)
					}
				})
			}).catch(()=>{})
	    },
		/**
		* 选择的批量数据
		* @param selection:传入批量数据对象
		*/
	    batchSelect(selection){
	    	var vm = this;
	    	vm.deviceSelection = selection;
	 	},
		exportDevice(){ // 导出
			var vm = this,
				params={
					timeZone:timeZone,
					group_id:vm.rowDataGroup.id
				},
				exportUrl ='';
			if(vm.deviceType == 'eNB'){
				exportUrl = '${ctx}/system/device/enodeb/exportENBCsvFile.action';
				params.like_fields = "serial_number";
			}else if(vm.deviceType == 'gNB'){
				exportUrl = '${ctx}/system/device/enodeb/exportENBCsvFile.action';
				params.isGnb = 1;
				params.like_fields = "serial_number";
			}else if(vm.deviceType == 'CPE'){
				exportUrl = '${ctx}/system/device/cpe/exportCPECsvFile.action';
				params.like_fields = "serial_number";
			}else if(vm.deviceType == 'WCG'){
				exportUrl = '${ctx}/egw/register/exportEGWToCSV.action';
			}
			params.search_text = vm.params_device.search_text;
			exportByForm(exportUrl,params);
		},
		// 修改设备
		modifyDevice(row){
			var vm = this,
				deviceType = vm.deviceType;

			vm.rowDataDevice = row;

			Object.keys(vm.modifyDeviceForm).forEach(function(key){
				if(row[key] == undefined && row[key] == null){
					vm.modifyDeviceForm[key] = vm.defaultModifyDeviceForm[key]
				}else{
					vm.modifyDeviceForm[key] = row[key];
				}
			});
			
			vm.rightBoxTitle = '<%=rb.getString("XiuGai")%>';
			vm.addGroupType = '';
			vm.rightBoxType = 'modify';
			vm.deviceListRightBoxShow = true;
            vm.showSubmitLoading = false;
			vm.$nextTick(function(){
				initForm(vm.$refs.modifyDeviceForm);
			})
			
		},
	    movecells(){ //批量操作 -- 移动设备到设备组
	    	var vm = this,
				idsList = [],
				typeCode={
					'eNB': 0,
					'gNB': 1,
					'CPE': 2,
					'WCG': 4
				},
				deviceType = vm.deviceType,
				randomCode = Math.random().toString();
    	    if(vm.deviceSelection.length <= 0)return
			if(vm.deviceSelection){
				vm.deviceSelection.map((item)=>{
					if(deviceType == 'eNB' || deviceType == 'gNB'){
						idsList.push(item[vm.valCodes[deviceType]] + '_' + item.product)	
					}else{
						idsList.push(item[vm.valCodes[deviceType]]);
					}
				});
			} 
			vm.moveDevice.code = idsList.join(',');
			vm.moveDevice.groupId = [] 
			vm.moveDevice.showMoveInfo = true;
			vm.moveDevice.dialogTitle = '<%=rb.getString("YiDongDaoSheBeiZu")%>';
			vm.moveDevice.windowWidth = '550px';
			vm.moveDevice.groupUrl = '${ctx}/system/deviceGroup/getDeviceGroupList.action?type='+ typeCode[deviceType] +'&no_group_id=' + vm.params_device.group_id + '&randomCode=' + randomCode;
			
		
	    },
		moveTrue(){ // 确定移动
			var vm = this,
				moveSubList = {
					'eNB':'${ctx}/system/deviceGroup/moveCellToDeviceGroup.action',
					'gNB':'${ctx}/system/deviceGroup/moveCellToDeviceGroup.action?isGnb=1',
					'CPE':'${ctx}/cell/CPE/moveCpeToDeviceGroup.action',
					'WCG':'${ctx}/egw/register/moveEGWToDeviceGroup.action'
				},
				url = moveSubList[vm.deviceType],
				params = {},
				str = '';
			
			params.toGroupId = vm.moveDevice.groupId
			params[vm.subCodes[vm.deviceType]] = vm.moveDevice.code;
			str = vm.moveDevice.groupId.toString();
			if(str !== ''){ // 如果没有选择设备组 提示先选择 只有在选择的时候才进行数据操作
				axios.post(url, stringify(params)).then(function(response){
					let data = response.data;
					if(data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});	
							vm.moveDevice.showMoveInfo = false;
							vm.$refs.ctableDevice.refresh();
							vm.$refs.ctableDevice.clearSelection();
						}else{
							vm.$message({
								type:'error',
								message:data.message,
							})
						}
				})
			}else{
				vm.$message({
					type:'error',
					message:'<%=rb.getString("QingXuanZeSheBeiZu")%>',
				})
			}
		
		},
		// 批量移入回收站设备
		recycleCells(){
			var vm = this,
	    		params = {},
				urlsCodes = {
					'eNB':'${ctx}/recycle/moveDeviceToRecycle.action',
					'gNB':'${ctx}/recycle/moveDeviceToRecycle.action?isGnb=1',
					'CPE':'${ctx}/recycle/moveCpeDeviceToRecycle.action',
				},
				subCodes={
					'eNB':'smallCellCodeStr',
					'gNB':'smallCellCodeStr',
					'CPE':'cpeCodeStr',
				}
	    		urls = urlsCodes[vm.deviceType],
				idsList = [],
				deviceType = vm.deviceType;
			if(vm.deviceSelection.length <= 0)return
			vm.deviceSelection.map((item,index) => {
				if(deviceType == 'eNB' || deviceType == 'gNB'){
					idsList.push(item[vm.valCodes[deviceType]]);
				}else{
					idsList.push(item[vm.valCodes[deviceType]]);
				}
			})
    	    params[subCodes[deviceType]] = idsList.join(',');
			var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueRenJiangSheBeiYiRuHuiShouZhan")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("YiRuHuiShouZhanTiShi")%>' +'</div>';
	    	vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
				customClass:"",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				dangerouslyUseHTMLString:true
			}).then(()=>{
				axios.post(urls,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						vm.$message({
					    	message: '<%=rb.getString("ChengGong")%>' ,
							type:'success',
						})
						vm.$refs.ctableDevice.refresh();
						vm.$refs.ctableDevice.clearSelection();
					}else {
						vm.$message.error(data.message)
					}

				}).catch(function(error){})

			}).catch(()=>{

			})
		},
		// 批量删除
	    deleteCells(){
	    	var vm = this,
	    		params = {},
				urlsCodes = {
					'eNB':'${ctx}/system/deviceGroup/delCellinfo.action',
					'gNB':'${ctx}/system/deviceGroup/delCellinfo.action?isGnb=1',
					'CPE':'${ctx}/cell/CPE/delCpeinfo.action',
					'WCG':'${ctx}/egw/register/delEGWInfo.action'
				}
	    		url = urlsCodes[vm.deviceType],
				idsList = [],
				deviceType = vm.deviceType;
			if(vm.deviceSelection.length <= 0)return
			vm.deviceSelection.map((item,index) => {
				if(deviceType == 'eNB' || deviceType == 'gNB'){
					idsList.push(item[vm.valCodes[deviceType]] + '_' + item.product)	
				}else{
					idsList.push(item[vm.valCodes[deviceType]]);
				}
			})
    	    params[vm.subCodes[deviceType]] = idsList.join(',');
	    
	    	vm.$confirm('<%=rb.getString("ShanChuSheBeiHeShuJu")%>','<%=rb.getString("QueRen")%>',{
				customClass:"",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning'
			}).then(()=>{
				axios.post(url,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						vm.$message({
					    	message: '<%=rb.getString("ChengGong")%>' ,
							type:'success',
						})
						vm.$refs.ctableDevice.refresh();
						vm.$refs.ctableDevice.clearSelection();
					}else {
						vm.$message.error(data.message)
					}
					
				}).catch(function(error){})
				
			}).catch(()=>{
				
			})
	    },
		isNumeric(str) {
			if(str.length==0){
				return false;
			}
			for(var i=0;i<str.length;i++){
				if(str.charAt(i)<"0" || str.charAt(i)>"9"){
					return false;
				}
			}
			return true;  
		},
		// 模糊搜索
		query(){
			var vm = this,
				deviceType = vm.deviceType,
				likeFields = {
					'eNB':'serial_number',
					'gNB':'serial_number',
					'CPE':'serial_number,macaddress',
					'WCG':'egwSn'
				};
			vm.params_device.search_text = this.search_text;
			vm.params_device.like_fields = likeFields[deviceType];
		},
		// 搜索域聚焦事件
		queryInputFocus(){
			var vm = this,
				deviceType = vm.deviceType,
				codes={
					'eNB':'<%=rb.getString("Title_SheBeiBianMa")%>',
					'gNB':'<%=rb.getString("Title_SheBeiBianMa")%>',
					'CPE':'<%=rb.getString("CPEXuLieHao")%>/<%=rb.getString("MACDiZhi")%>',
					'WCG':'<%=rb.getString("eGWBianMa")%>'
				};
			
			vm.placeholderText = codes[deviceType];
		},
		// 搜索域失焦事件
		queryInputBlur(){
			var vm = this;
			vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
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

			vm.$refs.ctableDevice.clearSelection();
		},
		// 设备已选表格 单个删除事件
		delBulkSelected(rows){
			var vm = this,
				deviceType = vm.deviceType,
				codes = {
					'eNB':'small_cell_code',
					'gNB':'small_cell_code',
					'CPE':'cpe_code',
					'WCG':'egwCode'
				},
				tabs = 'ctableDevice',
				rowKey = codes[deviceType];

			vm.deviceSelection = vm.deviceSelection.filter((items)=>{
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
		deviceTypeChange(type){
			var vm = this;
			vm.deviceType = type;
			vm.deviceUrl = vm.tabUrlList[vm.deviceType];
			vm.params_device.search_text = '';
			if(vm.rightBoxType == 'modify'){
				vm.rightBoxClose();
			}
			vm.$refs.ctableDevice.clearSelection();//刷新列表
		},
		// 导入证书文件关闭
		rightBoxClose(){
			var vm = this,
				type = vm.rightBoxType;
			vm.deviceListRightBoxShow = false;
			if(type == 'add'){
				vm.addDeviceForm.serialNumber = '';
				vm.typeFlag = true;
				vm.addSelectFlag = false;
				vm.addFileName = '';
				vm.$refs.addUpload.clearFiles();
				vm.rightBoxType = '';
			}else if(type == 'addGroup'){
				var params = {
					topGroupName:'',
					addSubCheck:false,
					subGroupName:'',
                    enable: '0',
                    matching_mode: 'deviceName',
                    name_contains: '',
                    tac_rag: '',
					desc:'',
					deviceType:[]
				};
				Object.assign(vm.addGroupForm,params);
				
				// 重置 nameContainsContentList 为默认状态
				vm.nameContainsContentList = [{
					condition: 'contain',
					value: '',
					hasError: false
				}];
				
				vm.addGroupDeviceList.map((item)=>{
					if(item.label == vm.deviceType){
						vm.$refs[item.refStr][0].clear();//刷新列表
					}
				})
			}else if(type == 'modify'){
                Object.keys(vm.modifyDeviceForm).forEach(function(key){
                    vm.modifyDeviceForm[key] = vm.defaultModifyDeviceForm[key]
                });
            }
		},
		// 新增设备提交
		addDeviceSubmit(){
			var vm = this,
				urls = '',
				snStr = vm.addDeviceForm.serialNumber || '',
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;}),
				deviceType = vm.addDeviceForm.deviceType,
				inputType = vm.addDeviceForm.inputType,
				message = '<%=rb.getString("TianJiaSheBeiChengGong")%>',
				params={
					group_id:vm.addDeviceForm.group_id,
					timeZone:timeZone
				};
            if(vm.showSubmitLoading)return
			// if(snStr && inputType == 'mac') {
			// 	snStr = snStr.split(';').map(function(item){
			// 		return (item.trim().toLocaleUpperCase().match(/[a-zA-Z0-9]{2}/g) || []).join(':');
			// 	}).join(';');
			// }
			
			if(deviceType == 'eNB'){
				urls = '${ctx}/system/deviceGroup/addAndAssignEnb.action';
				if(vm.isDeviceMoreParams == 'true'){
					Object.keys(vm.addDeviceForm).forEach(function(key){
						if(vm.addDeviceForm[key]){
							params[key] = vm.addDeviceForm[key];
						}
					});
					delete params.deviceType
					delete params.inputType
					delete params.selectType
					urls = '${ctx}/system/deviceGroup/addDevice.action';
				}else{
					params.serialNumber = list.join(";");
				}
				params.sns = list.join(";");
			}else if(deviceType == 'gNB'){
				urls = '${ctx}/system/deviceGroup/addAndAssignEnb.action';
				params.isGnb = 1;
				params.serialNumber = list.join(";");
			}else if(deviceType == 'CPE'){
				urls = '${ctx}/cell/CPE/addAndAssignCpe.action';
				if(inputType == 'mac'){
					params.macAddress = list.join(";");
				}else{
					params.sns = list.join(";");
				}
				params.type = inputType;
			}else if(deviceType == 'WCG'){
				urls = '${ctx}/egw/register/addDevice.action';
				params.serialNumber = list.join(";");
			}
			if(vm.addDeviceForm.selectType == '1'){
				if(vm.addFileName){
					vm.$refs.addUpload.submit();
				}else{
				    vm.typeFlag = true;
					vm.addSelectFlag = true;
				}
			}else{
				if(vm.siteEnable && vm.addDeviceForm.selectType == '0') {
					params.sub_station_name = vm.addDeviceForm.sub_station_name;
				}
				vm.$refs.addDeviceForm.validate((valid) => {
					if(valid){
                        vm.showSubmitLoading = true;
						axios.post(urls,stringify(params)).then(function(response){
							let data = response.data;
							if (data.success){
								//刷新表格,关闭新建设备弹窗
								vm.$refs.ctableDevice.refresh();//刷新列表
								vm.rightBoxClose();
								if(deviceType == 'CPE' && data.message){
									vm.$message({
										message: data.message,
										type:'success',
									})
								}else{
									vm.$message({
										message: message,
										type:'success',
									})
								}
							}else {
								vm.$message.error(data.message)
                                vm.showSubmitLoading = false;	
							}
						}).catch(function(error){})
						
					}else{
					}
				})
			}
		},
		addBeforeUpload(file){
			var vm = this, 
				urls = '',
				FileName = file.name,
				deviceType = vm.addDeviceForm.deviceType,
				inputType = vm.addDeviceForm.inputType,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};

			vm.isCSVFile = FileName.indexOf('.csv') >= 0;

			fd.append('uploadFile',file); //文件流
			fd.append('FileName',FileName);//文件名
			fd.append('group_id',vm.addDeviceForm.group_id);//文件名
			if(deviceType == 'eNB'){
				urls = '${ctx}/system/deviceGroup/uploadFile.action';
			}else if(deviceType == 'gNB'){
				urls = '${ctx}/system/deviceGroup/addAndAssignEnb.action';
				fd.append('isGnb',1);
				urls = '${ctx}/system/deviceGroup/uploadFile.action';
			}else if(deviceType == 'CPE'){
				if(inputType == 'mac'){
					urls = '${ctx}/cell/CPE/uploadFile.action?importType=append';
				}else{
					urls = '${ctx}/cell/CPE/uploadFile2.action?importType=append';
				}
			}else if(deviceType == 'WCG'){
				urls = '${ctx}/egw/register/uploadFile.action';
			}
			vm.showSubmitLoading = true;
			axios.post(urls,fd,config).then(function(res){
				let data = res.data;
                if(deviceType == 'eNB' || deviceType == 'gNB'){
                    // checkSuccess: true 标识导入全部成功；
                    if(data["checkSuccess"]){
						//checkSuccess: true 时，设备注册提示走原有的逻辑根据 success 字段处理，成功即成功，失败即失败
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							})
							//刷新表格,关闭新建设备弹窗
							vm.$refs.ctableDevice.refresh();//刷新列表
							vm.rightBoxClose();
						}else{
							vm.$message.error(data["msg"]);
                            vm.showSubmitLoading = false;
						}
						vm.importResultDialog = false;
                    }else{
                        // false 查询导入结果; 当前导入含有失败情况时；
                        vm.checkSuccessCount = data.checkSuccessCount;
                        vm.checkUnSuccessCount = data.checkUnSuccessCount;
                        vm.importResultDialog = true;
						vm.csvMsgTips = data['msg'];
                        vm.showSubmitLoading = false;
                    }
                }else{
                    //其它网元导入，逻辑不变
                    if(data["success"]){
                        if(deviceType == 'CPE' && data.msg){
                            vm.$message({
                                message: data.msg,
                                type:'success',
                            })
                        }else{
                            vm.$message({
                                message: '<%=rb.getString("ChengGong")%>',
                                type:'success',
                            })
                        }
                        //刷新表格,关闭新建设备弹窗
                        vm.$refs.ctableDevice.refresh();//刷新列表
                        vm.rightBoxClose();
                    }else{
                        vm.$message.error(data["msg"])
                        vm.showSubmitLoading = false;
                    }
                }
                
			})

			return false;
			
		},
		//导入文件结果-下载
        downloadFailClick(){
            var vm = this,
				deviceType = vm.addDeviceForm.deviceType,
				typeRef = {
					'eNB': 'enb',
					'gNB': 'gnb',
					'CPE': 'cpe',
					'WCG': 'wcg'
				},
				type = typeRef[deviceType];

            vm.importResultDialog = false;
            exportByForm('${ctx}/system/deviceGroup/downloadFailureFile.action', { type: type})
        },
		addCheckFile(res,file){    //发送请求，校验device文件内容 
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
				vm.$refs.ctableDevice.refresh();//刷新列表
				vm.closeFileSelect();
			}else{
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.addUpload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		// 移除导入文件
		closeAddFileSelect(){
			var vm = this;
			vm.addFileName = '';
			vm.typeFlag = true;
			vm.addSelectFlag = false;
			vm.$refs.addUpload.clearFiles();
		},
		// 选择文件
		addFileSelect(){  
			var vm =this;
			vm.$refs.addUpload.clearFiles();
			vm.$refs['addFile_up'].click();
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		addFileChange(file,fileList){ 
			var vm = this;
			vm.addSelectFlag = false;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.csv';
			vm.typeFlag = typeFlag;
			if(typeFlag){
               vm.addFileName = file.name;
            }else {
                vm.addFileName = '';
            }

		},
		exportAddTemplate(){ // 导出模板
	    	var vm = this,
				urls = '',
				deviceType = vm.addDeviceForm.deviceType,
				inputType = vm.addDeviceForm.inputType,
    	    	params = {};
			
			if(deviceType == 'eNB'){
				urls = '${ctx}/system/deviceGroup/downloadImportCellTemplate.action';
			}else if(deviceType == 'gNB'){
				urls = '${ctx}/system/deviceGroup/downloadImportCellTemplate.action';
				params.isGnb = 1;
			}else if(deviceType == 'CPE'){
				urls = '${ctx}/cell/CPE/downloadImportCpeTemplate.action';
				params.type = inputType;
			}else if(deviceType == 'WCG'){
				urls = '${ctx}/egw/register/downloadImportEGWTemplate.action';
			}
        	var bool = checkParams(params);
			if(!bool) return false;
        	exportByForm(urls,params)
	    },
		modifyDeviceSubmit(){
			var vm = this,
				deviceType = vm.deviceType,
				urls = '',
				isChanged = isFormChanged(vm.$refs.modifyDeviceForm),
				params = {};
			if(vm.showSubmitLoading)return
			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}
			vm.$refs.modifyDeviceForm.fields.map(function(field){
				if(Array.isArray(field.fieldValue)){
					
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						var editList=[],subList=[];
						if(val != orVal) {
							editList.push(field.prop);
						}
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
							
					}else if(field.fieldValue != field.reinitialValue) {
						params[field.prop] = field.fieldValue;
					};
				}
			});
			if(deviceType == 'eNB'){
				urls = '${ctx}/cell/topo/setLocationInfo.action';
				params.cell_code = vm.rowDataDevice.small_cell_code;
				params.device_status = vm.modifyDeviceForm.device_status;
				['longitude', 'latitude', 'height', 'sub_station_name', 'mechanical_downtilt', 'vertical_3dB_beam_width', 'horizontal_azimuth'].map(function(key){
					params[key] = vm.modifyDeviceForm[key];
				});
                if(vm.showSiteId == 'true'){
                    params.site_id = vm.modifyDeviceForm.site_id;
                }
				if(vm.isDeviceMoreParams == 'true' && vm.showOrHideCol != 'true'){
					Object.keys(vm.isMoreParams).forEach(function(key){
						params[key] = vm.modifyDeviceForm[key];
					});
				}
				if(vm.isDeviceMoreParams == 'true' && vm.showOrHideCol == 'true'){
					Object.keys(vm.isThailandTrueParams).forEach(function(key){
						params[key] = vm.modifyDeviceForm[key];
					});
				}
			}else if(deviceType == 'gNB'){
				['longitude', 'latitude', 'height'].map(function(key){
					params[key] = vm.modifyDeviceForm[key];
				});
				params.sub_station_name = vm.modifyDeviceForm.sub_station_name;
				params.device_status = vm.modifyDeviceForm.device_status;
				params.cell_code = vm.rowDataDevice.small_cell_code;
				urls = '${ctx}/cell/cpeinfos/updateIsFix.action';
			}else if(deviceType == 'CPE'){
				['longitude', 'latitude', 'height','distance'].map(function(key){
					params[key] = vm.modifyDeviceForm[key];
				});
				urls = '${ctx}/cell/topo/setLocationInfo.action';
				params.cell_code = vm.rowDataDevice.cpe_code;
			}else if(deviceType == 'WCG'){
				urls = '${ctx}/egw/register/updateEGWInfo.action';
				params.egwSn = vm.rowDataDevice.egwSn;
				['egwName','egwDescription','hardwareModel'].map((item)=>{
					if(item == 'egwName' || item == 'egwDescription'){
						params[item] = encodeURIComponent(vm.modifyDeviceForm[item])
					}else{
						params[item] = vm.modifyDeviceForm[item];
					}
				})
			}
            if(deviceType == 'eNB' || deviceType == 'gNB'){
                params.remark = vm.modifyDeviceForm.remark;
            }
			vm.$refs.modifyDeviceForm.validate((valid) => {
				if(valid){
                    vm.showSubmitLoading = true;
					axios.post(urls,stringify(params)).then(function(response){
						let data = response.data;
						if ( data.success ){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});	
							//刷新表格,关闭新建设备弹窗
							vm.$refs.ctableDevice.refresh();//刷新列表
							vm.rightBoxClose();
						}else {
							vm.$message.error(data.message);
                            vm.showSubmitLoading = false;
						}
					}).catch(function(error){})
				}else{
					return false;
				}
		    })
		},
		// 判断是否为空
		isNull(val){
			if(val === undefined || val === null || val === "") return true;
			else return false;
		},
		// 导出弹窗设备类型改变事件
		exportDeviceTypeChange(list){
			var vm = this;
			if(list&&list.length > 0){
				vm.deviceExportErrorShow = false;
			}
		},
		// 导出弹窗收起事件
		deviceExportPopoverHide(){
			var vm = this;
			vm.exportDeviceType = [];
			document.body.click();
		},
		// 新增弹窗 设备类型改变
		addDeviceTypeChange(type){
			var vm = this;
			vm.addDeviceForm.serialNumber = '';
			if(type != 'CPE'){
				vm.addDeviceForm.inputType = 'sn';
			}else{
				vm.addDeviceForm.inputType = 'mac';
			}
		},
		// 设备组弹窗 设备类型改变事件
		addGroupDeviceTypeChange(list){
			var vm = this;
			vm.addGroupDeviceList.map((item)=>{
				if(vm.addGroupForm.deviceType.includes(item.label)){
					item.isShow = true;
				}else{
					item.isShow = false;
					item.search_text = '';
					item.queryParams.search_text = item.search_text;
				}
			});
			if(!vm.addGroupForm.deviceType.includes(vm.addGroupDeviceActiveName)){
				vm.addGroupDeviceActiveName = vm.addGroupForm.deviceType[0] ? vm.addGroupForm.deviceType[0] : '';
			}
		},
		// 设备组弹窗 tabs改变事件
		addGroupDeviceTabsEdit(targetName,action){
			var vm = this;
			if(action == 'remove'){
				var deviceTypeList = vm.addGroupForm.deviceType;
				
				deviceTypeList = deviceTypeList.filter((items)=>{
					return items != targetName
				})
				vm.addGroupDeviceList.map((item)=>{
					if(item.label == targetName){
						item.isShow = false;
						item.search_text = '';
						item.queryParams.search_text = item.search_text;
					}
				});
				vm.addGroupForm.deviceType = deviceTypeList;
				if(!vm.addGroupForm.deviceType.includes(vm.addGroupDeviceActiveName)){
					vm.addGroupDeviceActiveName = vm.addGroupForm.deviceType[0] ? vm.addGroupForm.deviceType[0] : '';
				}
			}
		},
		// 设备组弹窗 表格数据批量选择
		addGroupSelectChange(selection,targetName){
			var vm = this;
		},
		addGroupTableQuery(targetName){
			var vm = this;
			vm.addGroupDeviceList.map((item)=>{
				if(item.label == targetName){
					item.queryParams.search_text = item.search_text;
				}
			});
		},
		// 设备组 一级设备组新增 子设备组新增，修改
		addGroupSubmit(){
			var vm = this,
				type = vm.addGroupType,
				urls = '',
				codes = {
					'eNB':'enbList',
					'gNB':'gnbList',
					'CPE':'cpeList',
					'WCG':'egwList'
				},
				params = {},
                moveDeviceGroupRule = {};
            if(vm.showSubmitLoading)return

            if(vm.deviceType != 'WCG' && vm.addGroupForm.enable == '1'){
                moveDeviceGroupRule.move_to_group_id = '';
                moveDeviceGroupRule.enable = vm.addGroupForm.enable;
                moveDeviceGroupRule.matching_mode = vm.addGroupForm.matching_mode;
                moveDeviceGroupRule.tac_rag = vm.addGroupForm.tac_rag;
                moveDeviceGroupRule.order = '';
                moveDeviceGroupRule.device_type = vm.deviceType.toUpperCase();
                
                // 使用 nameRuleList 传递过滤器数据
                moveDeviceGroupRule.nameRuleList = vm.nameContainsContentList.map(function(item){
                    return {
                        condition: item.condition,
                        value: item.value,
                        andOr: item.andOr
                    };
                });
            }
			if(type == 'addOneGroup'){
				urls='${ctx}/manage/deviceGroup/addTopGroupAndDevice.action';
				params.topGroupName = vm.addGroupForm.topGroupName;
				if(vm.addGroupForm.addSubCheck){
					params.subGroupName = vm.addGroupForm.subGroupName;
					params.desc = vm.addGroupForm.desc;
					vm.addGroupDeviceList.map((item)=>{
						if(item.label == vm.deviceType){
							var selectDeviceList = vm.$refs[item.refStr][0].getData();
							if(selectDeviceList.length>0){
								var selectCodeList = [];
								selectDeviceList.map((items)=>{
									if(item.label == 'eNB' || item.label == 'gNB'){
										selectCodeList.push(items[vm.valCodes[item.label]] + '_' + items.product)	
									}else{
										selectCodeList.push(items[vm.valCodes[item.label]]);
									}
								})
								params[codes[item.label]] = selectCodeList.join(',');
							}
						}
					});
				}
			}else if(type == 'addSubGroup'){
				urls='${ctx}/manage/deviceGroup/addSubGroupAndDevice.action';
				params.pid = vm.rowDataOneGroup.id,
				params.subGroupName = vm.addGroupForm.subGroupName;
				params.desc = vm.addGroupForm.desc;
				vm.addGroupDeviceList.map((item)=>{
					if(item.label == vm.deviceType){
						var selectDeviceList = vm.$refs[item.refStr][0].getData();
						if(selectDeviceList.length>0){
							var selectCodeList = [];
							selectDeviceList.map((items)=>{
								if(item.label == 'eNB' || item.label == 'gNB'){
									selectCodeList.push(items[vm.valCodes[item.label]] + '_' + items.product)	
								}else{
									selectCodeList.push(items[vm.valCodes[item.label]]);
								}
							})
							params[codes[item.label]] = selectCodeList.join(',');
						}
					}
				});
			}else if(type == 'editSubGroup'){
				urls='${ctx}/manage/deviceGroup/modSubDeviceGroupInfo.action';
				params.groupId = vm.rowDataGroup.id,
				params.subGroupName = vm.addGroupForm.subGroupName;
                if(vm.deviceType != 'WCG'){
                    moveDeviceGroupRule.move_to_group_id = vm.rowDataGroup.id;
                    moveDeviceGroupRule.id = vm.deviceRuleId;
                    moveDeviceGroupRule.enable = vm.addGroupForm.enable;
                    moveDeviceGroupRule.order = vm.deviceRuleOrder;
                }
				params.desc = vm.addGroupForm.desc;
				vm.addGroupDeviceList.map((item)=>{
					if(item.label == vm.deviceType){
						var selectDeviceList = vm.$refs[item.refStr][0].getData();
						if(selectDeviceList.length>0){
							var selectCodeList = [];
							selectDeviceList.map((items)=>{
								if(item.label == 'eNB' || item.label == 'gNB'){
									selectCodeList.push(items[vm.valCodes[item.label]] + '_' + items.product)	
								}else{
									selectCodeList.push(items[vm.valCodes[item.label]]);
								}
							})
							params[codes[item.label]] = selectCodeList.join(',');
						}
					}
				});
			}
            if(vm.deviceType != 'WCG'){
                params.moveDeviceGroupRule = JSON.stringify(moveDeviceGroupRule);
            }
			vm.$refs.addGroupForm.validate((valid) => {
				if(valid){
                    vm.showSubmitLoading = true;
					axios.post(urls,stringify(params)).then(function(response){
						let data = response.data;
						if ( data.success ){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});	
							vm.queryGroupList(vm.queryGroupSearchText);
							vm.rightBoxClose();
						}else {
							vm.$message.error(data.message);
                            vm.showSubmitLoading = false;
						}
					}).catch(function(error){})
				}else{
					return false;
				}
		    })

		},
		//一级设备组修改 提交
		oneGroupEditSubmit(node,data,ev){
			var vm = this, 
				url = '${ctx}/system/deviceGroup/modTopDevice.action',
				reg = /^[a-zA-Z0-9_\u4e00-\u9fa5,\s]{1,50}$/,
				params = {
					groupName:data.group_name,
					groupId: data.id
				};
			if(params.groupName && reg.test(params.groupName)){
				axios.post(url,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});
						vm.queryGroupList(vm.queryGroupSearchText);
					}else {
						vm.$message.error(data.message)
					}
				}).catch(function(error){})
			}else{
				vm.$message.error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>')
			}
			event.stopPropagation();
		},
		//一级设备组修改 取消
		oneGroupEditClose(){
			var vm = this;
			vm.queryGroupList(vm.queryGroupSearchText);
		},
		// 新增设备 输入类型切换
		addDeviceInputTypeChange(){
			var vm = this;
			vm.$refs.addDeviceForm.validateField('serialNumber');
		},
		// 打开设备回收站
		openDeviceRecycleBin(){
			var vm = this;
			vm.sharingSlideUrl = "${ctx}/recycle/toDeviceRecycleBinPage.action";
			//vm.sharingSlideUrl = "${ctx}/cell/fault/goAlarmDetail.action";
			vm.sharingSlideHeight = '100%';
			vm.sharingSlideWidth = '100%';
			vm.sharingSlidePosition = 'top';
			vm.sharingSlideFooter = false;
			vm.sharingSlideHeader = false;
			vm.sharingSlideTitle = 'Recycle Bin';
			vm.$refs.sharingSlide.showSlide(function(){
				eventBus.$emit('recycle-bin-init',vm.deviceType);
			});
		},
		// 关闭回收站页面
		sharingSlideCancel(){
			var vm = this;
			vm.$refs.sharingSlide.hide();
			vm.$refs.ctableDevice.refresh();//刷新列表
		},
	    toAddSite(evt) {
			var vm = this;

			Object.assign(vm.addSiteForm, {
				siteName: '',
				latitude: '',
				longitude: ''
			});
			vm.siteAddShow = true;
			vm.preventEvent(evt);
		},
		preventEvent(evt) {
			evt.stopPropagation();
		},
		addSiteSubmit() {
			var vm = this,
				url = '${ctx}/site/addSiteInfo.action';

			vm.$refs.addSiteForm.validate(function(r){
				if(r) {
					axios.post(url, stringify(vm.addSiteForm)).then(function(res){
						var data = res.data;
		
						if(data.success == true) {
							vm.querySiteNames();
							vm.siteAddShow = false;
							vm.$message({
								type: 'success',
								message: '<%=rb.getString("ChengGong")%>'
							});
						}else {
							vm.$message.error(data.message);
						}
					});
				}
			});
		},
        // 打开设备归属设备组规则页面
		configuraRule(){
			var vm = this;
			// vm.sharingSlideUrl = "${ctx}/cell/cpeinfos/toDeviceAttrRulePage.action";
            vm.sharingSlideUrl = "${ctx}/moveDeviceGroupRule/rule/toDeviceAttrRule.action";
			vm.sharingSlideHeight = '100%';
			vm.sharingSlideWidth = '100%';
			vm.sharingSlidePosition = 'top';
			vm.sharingSlideFooter = false;
			vm.sharingSlideHeader = false;
			vm.$refs.sharingSlide.showSlide();
		},
        // 设备归属设备组类型切换 deviceName/tac
        matchingModeChange(){
            var vm = this;
            vm.addGroupForm.tac_rag = '';
            vm.addGroupForm.name_contains = '';
            vm.$refs.addGroupForm.clearValidate('tac_rag');
            vm.$refs.addGroupForm.clearValidate('name_contains');
            
            // 重置 nameContainsContentList 为默认状态
            vm.nameContainsContentList = [{
                condition: 'contain',
                value: '',
                hasError: false
            }];
        },
        // 查看二级设备组规则信息
        getDeviceGroupRuleInfo(){
            var vm = this,
                urls = '${ctx}/moveDeviceGroupRule/rule/info/getByMoveToGroupId.action',
                params = {
                    move_to_group_id: vm.rowDataGroup.id,
                    device_type: vm.deviceType.toUpperCase()
                };
            axios.post(urls,stringify(params)).then(function(response){
                var data = response.data;
                if(data){
                    vm.addGroupForm.enable = data.enable ? data.enable : '0';
                    vm.addGroupForm.matching_mode = data.matching_mode ? data.matching_mode : 'deviceName';
                    vm.addGroupForm.tac_rag = data.tac_rag ? data.tac_rag : '';
                    vm.deviceRuleOrder = data.order ? data.order : '';
                    vm.deviceRuleId = data.id ? data.id : '';
                    
                    // 从接口返回的 nameRuleList 回显到 nameContainsContentList
                    if(data.nameRuleList && data.nameRuleList.length > 0){
                        vm.nameContainsContentList = data.nameRuleList.map(function(item){
                            return {
                                condition: item.condition,
                                value: item.value,
                                andOr: item.andOr,
                                hasError: false
                            };
                        });
                    } else {
                        // 如果没有 nameRuleList，添加一个默认的空过滤器
                        vm.nameContainsContentList = [{
                            condition: 'contain',
                            value: '',
                            hasError: false
                        }];
                    }
                }
            })
        },
        //TAC验证
        isValidIntegerTacRange(val, min, max) {
            var vm = this;
            // 将输入字符串按逗号分割成多个部分
            const parts = val.split(',');

            for (let part of parts) {
                // 检查是否有连字符
                if (part.includes('-')) {
                    const range = part.split('-');
                    const start = parseInt(range[0], 10);
                    const end = parseInt(range[1], 10);

                    // 检查范围内的数字是否合法
                    if (isNaN(start) || isNaN(end) || !vm.isNumeric(start) || !vm.isNumeric(end) || start >= end || start < min || end > max) {
                        return false;
                    }
                } else {
                    const num = parseInt(part, 10);

                    // 检查单个数字是否合法
                    if (isNaN(num) || num < min || num > max) {
                        return false;
                    }
                }
            }

            return true;
        },
		activeWithLocation() {
			var vm = this,
				params = {};

			Object.assign(params, vm.locationParams);
			$.post("${ctx}/system/deviceGroup/updateDetectionEnable.action", params, function(data){
				if(data["success"]){
					showMsg('success_msg','<%=rb.getString("XiaFaChengGong")%>');
					vm.$refs.ctableDevice.refresh();
					vm.$refs.ctableDevice.clearSelection();
					vm.locationReasonableShow = false;
				}else{
					showMsg('error_msg',data["message"]);
				}
			},"json")
		},
		setDetectEnable(row) {
			var vm = this,
				active = row.isCheckInActive,
				params = {
					ids: row.serial_number,
					detectionStatus: '0'
				};

			vm.currentRow = row;

			var confirmMsg = '<%=rb.getString("QueRenGuanBi")%>',
                activeTips = '',
                h = vm.$createElement;

			if(active != '1') {
				confirmMsg = '<%=rb.getString("QueRenKaiQi")%>';
				activeTips = '<%=rb.getString("SheBeiJingWeiDuJianCeTiShi1")%> ' + latitudeToleranceRange + ' <%=rb.getString("SheBeiJingWeiDuJianCeTiShi2")%>';
				params.detectionStatus = '1';
				
				// same with enb monitor
				//confirmMsg = '<%=rb.getString("JiHuoQueRen")%>';
				activeTips = '<%=rb.getString("WeiZhiJianCeBianHuaTi")%>';

				vm.locationTips = confirmMsg;
				vm.locationReasonableShow = true;
				Object.assign(vm.locationParams, params);
				return;
			}

			vm.$msgbox({
				title: '<%=rb.getString("QueRen")%>',
				message: h('p', null, [
					h('div', null, confirmMsg),
					h('i', { style: 'color: #7A7992'}, activeTips)
				]),
				showCancelButton: true,
				customClass: 'warningConfirm',
				confirmButtonText: '<%=rb.getString("QueDing")%>',
				cancelButtonText: '<%=rb.getString("QuXiao")%>',
				type: 'warning',
				closeOnClickModal: false
			}).then((r)=>{
				if(r == 'confirm') {
					$.post("${ctx}/system/deviceGroup/updateDetectionEnable.action", params, function(data){
						if(data["success"]){
							showMsg('success_msg','<%=rb.getString("XiaFaChengGong")%>');
							vm.$refs.ctableDevice.refresh();
							vm.$refs.ctableDevice.clearSelection();
						}else{
							showMsg('error_msg',data["message"]);
						}
					},"json")
				}
			})
		},
		batchSetDetectEnable() {
			var vm = this,
				params = {
					serial_number: '',
					isCheckInActive: '0'
				};

			if(vm.deviceSelection.length <= 0) return;

			var codeList = vm.deviceSelection.map(function(item){
					return item.serial_number;
				});
			params.serial_number = codeList.join(',');

			vm.setDetectEnable(params);
		},
		batchSetDetectDisable() {
			var vm = this,
				params = {
					serial_number: '',
					isCheckInActive: '1'
				};

			if(vm.deviceSelection.length <= 0) return;

			var codeList = vm.deviceSelection.map(function(item){
					return item.serial_number;
				});
			params.serial_number = codeList.join(',');

			vm.setDetectEnable(params);
		},
        // 添加过滤器
		addNameContainsContent(){
			var vm = this;
			if(vm.nameContainsContentList.length < 10){
				// 检查前面是否有选择了 Or 的行
				var hasOrCondition = vm.nameContainsContentList.some(function(filter, index){
					return index > 0 && filter.andOr === 'or';
				});
				
				// 如果已有 Or，新增行默认为 Or；否则默认为 And
				var defaultAndOr = hasOrCondition ? 'or' : 'and';
				
				vm.nameContainsContentList.push({
					andOr: defaultAndOr,
					condition: 'contain',
					value: '',
					hasError: false
				});
			}
		},
		// 动态获取 And/Or 选项（Or下面不能出现And，Or上面相邻的And能改成Or）
		getAndOrOptions(index){
			var vm = this;
			
			// 检查当前行之前是否有选择了 Or 的行
			var hasOrBefore = false;
			for(var i = 1; i < index; i++){
				if(vm.nameContainsContentList[i].andOr === 'or'){
					hasOrBefore = true;
					break;
				}
			}
			
			// 检查下一行（相邻）是否为 And
			var nextIsAnd = false;
			if(index + 1 < vm.nameContainsContentList.length){
				if(vm.nameContainsContentList[index + 1].andOr === 'and'){
					nextIsAnd = true;
				}
			}
			
			// 如果之前有 Or，则禁用 And 选项
			// 如果下一行是 And，则禁用 Or 选项（Or上面相邻的And能改成Or，但Or下面不能是And）
			return vm.conditionAndOrOptions.map(function(option){
				return {
					label: option.label,
					value: option.value,
					disabled: (hasOrBefore && option.value === 'and') || (nextIsAnd && option.value === 'or')
				};
			});
		},
		// 删除过滤器
		removeFilter(index){
			var vm = this;
			if(vm.nameContainsContentList.length > 1){
				vm.nameContainsContentList.splice(index, 1);
				
				// 如果删除的是第一行，需要移除新的第一行的 andOr 字段
				if(index === 0 && vm.nameContainsContentList.length > 0){
					vm.$delete(vm.nameContainsContentList[0], 'andOr');
				}
			}
		},
		// 验证单个输入框
		validateSingleInput(index){
			var vm = this;
			var filter = vm.nameContainsContentList[index];
			
			if(vm.addGroupForm.enable == '1' && vm.addGroupForm.matching_mode == 'deviceName'){
				// 检查是否为空或格式不正确
				if(!filter.value || filter.value.trim() === ''){
					vm.$set(filter, 'hasError', true);
				} else {
					vm.$set(filter, 'hasError', false);
				}
			} else {
				vm.$set(filter, 'hasError', false);
			}
			
            vm.$refs.addGroupForm.validateField('name_contains');
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
        // remark列label自定义方法
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
					{name: 'gsmvm', instance: typeof gsmvm !== 'undefined' ? gsmvm : null},
					{name: 'gnbMonitor', instance: typeof gnbMonitor !== 'undefined' ? gnbMonitor : null},
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
		}
	},
	computed:{
		nowInputNameContainsContent() {
			var vm = this,
				orGroups = [[]]; // 按 or 分组，初始有一个空组
			
			// 按 or 分组
			vm.nameContainsContentList.forEach(function(filter, index) {
				if (filter.value && filter.value.trim() !== '') {
					// 如果当前行是 or，创建新组
					if (index > 0 && filter.andOr === 'or') {
						orGroups.push([]);
					}
					
					// 将当前过滤器添加到最后一个组
					orGroups[orGroups.length - 1].push({
						condition: filter.condition,
						value: filter.value.trim()
					});
				}
			});
			
			// 过滤掉空组
			orGroups = orGroups.filter(function(group) {
				return group.length > 0;
			});
			
			if (orGroups.length === 0) {
				return '';
			}
			
			// 生成最终字符串
			var groupParts = [];
			orGroups.forEach(function(group) {
				var conditionParts = [];
				
				group.forEach(function(item) {
					// 查找 condition 对应的动词
					var verb = '';
					if (item.condition === 'contain') {
						verb = 'contain';
					} else if (item.condition === 'notContain') {
						verb = 'not contain';
					} else if (item.condition === 'startWith') {
						verb = 'start with';
					} else if (item.condition === 'endWith') {
						verb = 'end with';
					}
					
					conditionParts.push(verb + ' "' + item.value + '"');
				});
				
				// 组内用 and 连接
				var groupText = conditionParts.join(' and ');
				
				// 如果有多个 or 组，每个组都加括号；如果只有一个组但有多个条件，也加括号
				if (orGroups.length > 1 || group.length > 1) {
					groupText = '(' + groupText + ')';
				}
				
				groupParts.push(groupText);
			});
			
			// 组间用 or 连接
			var result = groupParts.join(' or ');
			
			// 如果有多个组或第一组有多个条件，添加 Must 前缀
			if (orGroups.length > 1 || orGroups[0].length > 1) {
				result = 'Must ' + result;
			} else {
				// 单个条件，首字母大写
				result = result.charAt(0).toUpperCase() + result.slice(1);
			}
			
			return result;
		},
		limitBatch(){
			return this.batchOperation ? '' : 1;
		},
		activeNeType() {
            var activeTab = sysMain.$refs.nav.editableTabs.filter((item)=>{
                return ['400001','400002','400003','400004'].includes(item.id);
            })[0].netType;

			return activeTab;
		},
		groupWritable() {
			var vm = this,
				write = true;
			
			if(vm.rowDataGroup && vm.rowDataGroup.write != '1') {
				write = false;
			}
			
			return write;
		},
		selectedDialogTitle() {
			var vm = this,
				deviceType = vm.deviceType
				codes = {
					'eNB':'<%=rb.getString("Title_SheBeiBianMa")%>',
					'gNB':'<%=rb.getString("Title_SheBeiBianMa")%>',
					'CPE':'<%=rb.getString("MACDiZhi")%>+<%=rb.getString("CPEBianMa")%>',
					'WCG':'<%=rb.getString("eGWBianMa")%>'
				};

			return codes[deviceType];
		},
		deviceTableRowKey() {
			var vm = this,
				deviceType = vm.deviceType
				codes = {
					'eNB':'small_cell_code',
					'gNB':'small_cell_code',
					'CPE':'cpe_code',
					'WCG':'egwCode'
				};

			return codes[deviceType];
		},
		optDeviceShow() {
			var vm = this,
				deviceType = vm.deviceType
				codes = {
					'eNB':'CODE_ENB_DEVICE_REGISTER',
					'gNB':'CODE_GNB_DEVICE_REGISTER',
					'CPE':'CODE_CPE_DEVICE',
					'WCG':'CODE_EGW'
				};

			return writableMap[codes[deviceType]] == true;
		},
		siteIdLabel(){
			return siteIdLabel
		},
		siteNameLabel(){
			return siteNameLabel
		},
		isCurrentTab() {
			var vm = this,
				tabId = vm.$el.parentNode.id.replace('tab_content_', '');

			return tabId == sysMain.$refs.nav.editableTabsValue;
		}
	},
	watch: {
		activeNeType() {
            if(this.rightBoxType){
                this.rightBoxClose();
            }
            this.init();
            this.$nextTick(function(){
                this.$refs.ctableDevice.clearSelection();
            });
		},
	},
	mounted(){
		var vm = this;
		vm.init();
	}
	
});

</script>