<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp"%>

<style>
    .select-80px .el-input__inner,
    .select-height-26  .el-input__inner{
        max-height: 26px;
        height: 26px;
    }
    .select-80px .el-input {
        width: 80px;
    }
    .item-cls {
        display: inline-block;
        margin-right: 40px;
        width: 45%;
    }
    .special-form {
        padding: 20px;
        flex: 1 auto;
        overflow: auto;
    }
    .special-form .el-form-item__label {
        text-align: left;
        line-height: 25px;
    }
    .prefix-title {
        font-weight: bold;
        padding: 3px 25px 0 0;
    }
    .bt-cls {
        padding: 2px 10px;
        border-radius: 3px;
        border: 1px solid #4d84ff;
        cursor: pointer;
    }
	.el-input-group__append{
		border-radius:0px;
		border-right:none;
	}
	.validate-item .el-input__inner{
		width:200px;
	}
	.validate-item .el-input-group__append{
		border:none;
		background:none;
	}
	.validate-item .el-form-item__error{
		display:none;
	}
	.is-error .el-input-group__append{
		color:#FA5555;
	}
	.validate-item .el-form-item__content{
		margin-left: 0px!important;
	}
	.validate-item .el-form-item__label{
		float: none;
	}
	#special_tab .mmeIpBox {
		margin-bottom: 5px;
	}
	#special_tab .mmeIpBox .el-input-group__append{
		background-color: #FFFFFF;
		border-right: 1px solid #dcdfe6;
		padding: 0 10px;
	}
	#special_tab .mmeIpBox .mmeIpErrCls{
		color: #FA5555;
		font-size: 12px;
		margin-left: 20px;
	}
	#special_tab .mmeIpBox .mmeIpMessageCls{
		color: #909399;
		font-size: 12px;
		margin-left: 20px;
	}
	#special_tab .cellListBox{
		width: 100%;
	}
	#special_tab .cellListBox  .cellListTitleBox{
		display: flex;
		align-items: center;
		position: relative;
	}
	#special_tab .buttonBoxCls{
		height: 28px;
		width: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
		border:1px solid #DFE2EE;
		border-radius: 8px;
		margin-left: 10px
	}
	#special_tab .buttonBoxCls .el-icon::before{
		color: #7A7992;
		font-size: 16px;
	}
	#special_tab .buttonBoxCls:hover{
		background-color: #F5F7FA;
	}
	#special_tab .cellListBox  .cellListTitleBox div:nth-child(3){
		margin-left: 10px;
		font-size: 12px;color:#999999;
	}
	#special_tab .cellListBox  .cellListTitleBox .rightButtonBox{
		display: flex;
		position: absolute;
		right:0px;
	}
	#special_tab .cellListBox .cellListMainBox{
		width: 100%;
	}
	#special_tab .cellListBox .cellListMainBox .cellItemCls{
		width: 100%;
		display: flex;
		border: 1px solid #DFE2EE;
		border-radius: 10px;
		box-sizing: border-box;
        background-color: #FFFFFF;
		margin: 10px 0px;
	}
	#special_tab .cellListBox .cellListMainBox .cellItemCls > div{
		height: 54px;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.haloDModeCodeCls{
		display: inline-block;
		width: 16px;
		height: 16px;
		line-height: 16px;
		text-align: center;
		font-size: 12px;
		border-radius: 2px;
		color: #FFFFFF;
		background-color: #73B8FF;
		margin-left: 3px;
	}
	#special_tab .itemLabelCls{
		font-size: 14px;
		color:#999999;
		margin-right: 10px;
	}
	#special_tab  .itemValueCls{
		display:inline-block;
		overflow:hidden;
		text-overflow:ellipsis;
		word-break:break-all;
		white-space: nowrap;
	}
	#special_tab  .activeCls{
		border:1px solid #4D84FF!important;
        background-color: #F4F9FF!important;
	}
	.tooltipCls.is-dark{
		background : #959595 ;
		color : #FFFFFF ;
	}
	.tooltipCls[x-placement^=top] .popper__arrow ,
	.tooltipCls[x-placement^=top] .popper__arrow::after{
		border-top-color: #959595!important;
	}

	.tooltipCls[x-placement^=bottom] .popper__arrow ,
	.tooltipCls[x-placement^=bottom] .popper__arrow::after {
		border-bottom-color: #959595!important;
	}
	.tooltipCls[x-placement^=right] .popper__arrow ,
	.tooltipCls[x-placement^=right] .popper__arrow::after {
		border-right-color: #959595!important;
	}
	.tooltipCls[x-placement^=left] .popper__arrow ,
	.tooltipCls[x-placement^=left] .popper__arrow::after {
		border-left-color: #959595!important;
	}
	.orangeIcon::before{
		font-size: 18px;
		color: #FF9D47!important;
	}
	#special_tab .mmeIpListBoxCls{
		width:340px;
		min-height: 34px;
		max-height: 96px;
		overflow: auto;
		border: 1px solid #DFE2EE;
		box-sizing: border-box;
		border-radius: 4px;
		padding: 2px 3px;
		margin-bottom: 20px;
	}
	#special_tab .mmeIpItemCls{
		height: 30px;
		width: 320px;
        box-sizing: border-box;
		display: flex;
		align-items: center;
		justify-content: space-between;
		box-sizing: border-box;
		border-radius: 100px;
		padding-left: 20px;
		font-size: 12px;
	}
	#special_tab .mmeIpItemCls:hover{
		background-color: #F4F9FF;
	}
	#special_tab .mmeIpItemCls:hover span:nth-child(2){
		display: block!important;
	}
	.closeSmallIcon::before{
		font-size: 16px;
	}
	.haloXIconBox{
		display: inline-block;
		height: 16px;
		width: 16px;
		line-height: 16px;
		text-align: center;
		font-size: 12px;
		font-weight: 600;
		color:#FFFFFF;
		background-color: #000;
	}
	#addCellDialog{
		z-index: 6666;
	}
	#addCellDialog .headQueryBox{
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	#addCellDialog .headQueryBox .el-input.el-input--small{
		width: 250px;
	}
	#addCellDialog .el-dialog__body{
		padding: 10px 20px 15px;
	}
	#addCellDialog  .el-ctable > div:nth-child(2){
		border: 1px solid #E9E9E9;
	}
	#addCellDialog  .el-pagination{
		border: 1px solid #E9E9E9;
		border-top:none;
		border-radius: 0px 0px 10px 10px; 
	}
	#addCellDialog  .dialogFooterCls{
		text-align: left;
		border-top:1px solid #E9E9E9;
		height: 60px;
		line-height: 60px;
		padding-bottom: 0px;
	}
    #addCellDialog .addErrorCls{
        color: #FA5555;
    }
    .halodModeFormItemCls .el-radio.is-bordered.is-checked{
        border-color: #409EFF!important;
    }
    .halodModeFormItemCls .el-radio__input.is-checked+.el-radio__label{
        color: #409EFF!important;
    }
    .halodModeFormItemCls .el-radio__input.is-checked .el-radio__inner {
        background-color: #4D84FF!important;
        border-color: #4D84FF!important;
    }
</style>
<div id="special_tab" class="container" style="display: flex; flex-direction: column;background: #fff;border-radius: 5px;">
    <el-form ref="form" :model="form" :rules="rules" label-width="200" label-position="top" class="special-form">
        <div v-show="hasKey('lbtModel')" class="group" label="LBT">
            <el-form-item label="<%=rb.getString("JiZhanMoShi")%>" class="item-cls" v-show="hasKey('lbtModel')" prop="lbtModel">
                <el-select v-model="form.lbtModel">
                    <el-option label="LTE" value="0"></el-option>
                    <el-option label="LBT" value="1"></el-option>
                    <el-option label="INITAL" value="2"></el-option>
                </el-select>
            </el-form-item>

            <el-form-item label="<%=rb.getString("LBTChuFaFangShi")%>" class="item-cls" v-show="hasKey('lbtTrigger')" prop="lbtTrigger">
                <el-select v-model="form.lbtTrigger">
                    <el-option label="period trigger" value="1"></el-option>
                    <el-option label="dedicated trigger" value="2"></el-option>
                    <el-option label="Both" value="3"></el-option>
                </el-select>
            </el-form-item>

            <el-form-item label="<%=rb.getString("LBTChuFaShiJian")%>" class="item-cls" v-show="hasKey('lbtTime')" prop="lbtTime">
                <el-input v-model="form.lbtTime"></el-input>
            </el-form-item>

            <el-form-item label="<%=rb.getString("ChongShiJianGe")%>" class="item-cls" v-show="hasKey('lbtRetryPeriod')" prop="lbtRetryPeriod">
                <el-input type="number" v-model="form.lbtRetryPeriod"></el-input>
            </el-form-item>

            <el-form-item label="LBT Periodic Search1 (T1)" class="item-cls" v-show="hasKey('lbtPeriodSearch1')" prop="lbtPeriodSearch1">
                <el-input v-model="form.lbtPeriodSearch1"></el-input>
            </el-form-item>

            <el-form-item label="LBT Periodic Search2 (T2)" class="item-cls" v-show="hasKey('lbtPeriodSearch2')" prop="lbtPeriodSearch2">
                <el-input v-model="form.lbtPeriodSearch2"></el-input>
            </el-form-item>
        </div>

        <div v-show="showHalob && hasKey('halobSwitch') && !showHaloD" class="group" label="HaloB">
            <el-form-item label="HaloB Enable" v-show="hasKey('halobSwitch')" prop="halobSwitch">
                <el-switch v-model="form.halobSwitch"
                    active-value="1"
                    inactive-value="0"></el-switch>
            </el-form-item>

            <el-form-item label="<%=rb.getString("LicenseMoShi")%>" class="item-cls" v-show="hasKey('halobModel')" prop="halobModel">
                <el-select v-model="form.halobModel">
                    <el-option label="--" value="0"></el-option>
                    <el-option label="Centralized" value="1"></el-option>
                    <el-option label="Single" value="2"></el-option>
                </el-select>
            </el-form-item>

            <el-form-item label="<%=rb.getString("QianYueYouXiaoShiChang")%>" class="item-cls" v-show="hasKey('halobPeriod') && form.halobModel != '2'" prop="halobPeriod">
                <el-select v-model="form.halobPeriod">
                    <el-option label="1Day" value="1"></el-option>
                    <el-option label="2Day" value="2"></el-option>
                    <el-option label="3Day" value="3"></el-option>
                    <el-option label="5Day" value="5"></el-option>
                    <el-option label="7Day" value="0"></el-option>
                    <el-option label="10Day" value="10"></el-option>
                </el-select>
            </el-form-item>
        </div>
        
        <div v-show="showHaloD" class="group" label="HaloD">
            <el-form-item label="<%=rb.getString("GongZuoMoShi")%>"  prop="" labei-width="140px" class="halodModeFormItemCls">
                <el-radio-group v-model="halodMode" size="small" :disabled="true">
                    <el-radio label="0" style="margin-right:20px;" border><%=rb.getString("ZhuZhan")%></el-radio>
                    <el-radio label="1" style="margin-right:20px;" border><%=rb.getString("CongZhan1")%></el-radio>
                    <el-radio label="2" border><%=rb.getString("CongZhan2")%></el-radio>
                </el-radio-group>
            </el-form-item>
			<div v-if="false">
				<el-form-item label="<%=rb.getString("JiZhanBenDiSCTPDuanKou")%>" class="item-cls validate-item" prop="sctpPort">
					<el-input v-model.trim='form.sctpPort'>
						<template slot="append"><%=rb.getString("FanWei")%>：1024~65535,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item label="MME Port" class="item-cls validate-item" prop="mmePort">
					<el-input v-model.trim='form.mmePort'>
						<template slot="append"><%=rb.getString("FanWei")%>：1024~65535,Integer</template>
					</el-input>
				</el-form-item>

                <div v-show="showMME">
                    <!--
                    <el-form-item label="MME IP" style='margin-bottom:0px;width: 610px;' :class="mmeCls" label-position="top" prop="mmeIp">
                        <el-input v-model='mmeVal'>
                            <template slot="append">PLMN</template>
                        </el-input>
                        <el-select style='vertical-align:bottom;margin-left:-3px;' class='mmeSelect' v-model='mme_plmn'>
                            <el-option v-for="item in plmnGroup" :label="item" :value="item"></el-option>
                        </el-select>
                        <span v-if="mmeGroup.length<16" class='el-icon el-icon-plus' style='vertical-align:middle;margin-left:1px;' @click='addMME("mme_plmn")'></span>
                        <span class='item-tip'>No more than 16,Not repeat</span>
                    </el-form-item>
                    <div style='overflow:auto;margin-bottom:20px;'>
                        <el-form-item class='suffixItem' v-for='(domain,index) in mmeGroup' style='width:250px;margin-bottom:8px;'>
                            <div class='form-suffix' style='min-width:235px;'>
                                <span class='text' style='min-width:100px;border-right:1px solid #A0C4F9;padding:6px 0px;height:10px;'>{{domain.mme}}</span>
                                <span class='text' style='min-width:100px;'>PLMN:<span>{{domain.plmn}}</span></span>
                                <span style='font-size:14px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click='removeMME(index)'></span>
                            </div>
                        </el-form-item>
                        <el-form-item prop="mmeIp">
                            <el-input v-model="form.mmeIp" class="hide-input" style="border:none;width: 300px;"></el-input>
                        </el-form-item>
                    </div>
                    -->
                    <el-form-item label="Halod IP"  prop="mmeIp" 
                        prop="mmeIp"
                        key="mmeIp"
                        :rules="{
                            validator: function(rule, val, cb){
                                if(showMME) {
                                    if(val && isValidIP(val)) {
                                        cb();
                                    }else {
                                        cb('Example:1.1.1.1');
                                    }
                                }else {
                                    cb();
                                }
                            }
                        }">
                        <el-input v-model.trim='form.mmeIp'></el-input>
                    </el-form-item>
                </div>

                <div v-show="!showMME" style="padding-left: 16px;">
                    <el-form-item label=""  prop="mmeIp" v-show="false">
                        <el-input v-model.trim='form.mmeIp'></el-input>
                    </el-form-item>
                    <div class="mmeIpBox">
                        <div>MME IP</div>
                        <el-input v-model.trim='mmeIP'>
                            <div slot="append" @click="addMmeIP"><span class="el-icon el-icon-plus"></span></div>
                        </el-input>
                        <span :class="mmeIpErr ? 'mmeIpErrCls' : 'mmeIpMessageCls' ">Example:1.1.1.1</span>
                    </div>
                    <div class="mmeIpListBoxCls">
                        <div v-for="item in mmeIpList" class="mmeIpItemCls">
                            <span>{{item}}</span>
                            <span class="el-icon el-icon-circle-close closeSmallIcon" @click="delHalodMmeIp(item)" style="display:none"></span>
                        </div>
                    </div>
                </div>
			</div>
			<div class="cellListBox">
				<div class="cellListTitleBox">
					<div style="display: flex;align-items: center;">
						<span style="margin-right:5px;">Cell List</span>
                        <div v-if="!lockDis" class="buttonBoxCls"  @click="lockTypeChange">
                            <span v-if="lockStatus == '1'" class="el-icon el-icon-operation-lock orangeIcon"></span>
						    <span v-if="lockStatus == '0'" class="el-icon el-icon-status-unlock orangeIcon"></span>
                        </div>
						<div v-if="lockDis" class="buttonBoxCls disabled">
                            <span v-if="lockStatus == '1'" class="el-icon el-icon-operation-lock orangeIcon"></span>
						    <span v-if="lockStatus == '0'" class="el-icon el-icon-status-unlock orangeIcon"></span>
                        </div>
					</div>
					<div class="rightButtonBox">
						<el-tooltip popper-class="tooltipCls" content="<%=rb.getString("ZiDongTanCeGengXin")%>" placement='bottom'>
							<div class="buttonBoxCls" @click="refrshCellList"><span class="el-icon el-icon-operation-scan"></span></div>
						</el-tooltip>
						<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("TianJia")%>' placement='bottom'>
							<div v-if="!lockDis" class="buttonBoxCls" @click="addCellClick"><span class="el-icon el-icon-plus "></span></div>
						</el-tooltip>
						<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("TianJia")%>' placement='bottom'>
							<div v-if="lockDis" class="buttonBoxCls disabled"><span class="el-icon el-icon-plus "></span></div>
						</el-tooltip>
					</div>
				</div>
				<div v-if="false" style="padding:5px 0px;font-size: 12px;color:#999999;">
					<%=rb.getString("HaloDGuanXiSuoDingTiShi")%>
                    <div><%=rb.getString("HaloDSuoDingJinYongTiShi")%></div>
				</div>
				<div class="cellListMainBox">
					<el-form-item label="" class="validate-item"  prop="">
						<div v-for="item in cellList" :class="item.form == halodMode ?  'cellItemCls activeCls' : 'cellItemCls'">
							<div style="width:40px">
								<span v-if="item.form !== '0' && item.serialNumber !== serialNumber" @click="cellItemDel(item)"  class="el-icon el-icon-operation-delete"></span>
							</div>
                            <div style="width:200px;justify-content: left;">
                                <span class="itemValueCls">{{item.serialNumber}}</span>
                                <span class="haloDModeCodeCls">{{item.form}}</span>
                            </div>
                            <div style="width:260px">
                                <span class="itemLabelCls" style="width:70px;"><%=rb.getString("HostName")%></span>
                                <el-tooltip popper-class="tooltipCls" :content='item.cellName' placement='top'>
                                    <span class="itemValueCls" style="width:170px;">{{item.hostName}}</span>
                                </el-tooltip>
                            </div>
                            <div style="width:160px">
                                <span class="itemLabelCls" style="width:50px;"><%=rb.getString("EnodebId")%></span>
                                <span class="itemValueCls" style="width:80px;">{{item.enodebId}}</span>
                            </div>
                            <div style="width:120px">
                                <span class="itemLabelCls" style="width:50px;">cell ID</span>
                                <span class="itemValueCls" style="width:40px;">{{item.cellId}}</span>
                            </div>
                            <div style="width:260px">
                                <span class="itemLabelCls" style="width:60px;">WAN IP</span>
                                <span class="itemValueCls" style="width:160px;">{{item.cellWanIp}}</span>
                            </div>
						</div>
            		</el-form-item>
				</div>
			</div>
        </div>
    </el-form>
    <div style="padding: 15px 0 15px 30px;border-top: 1px solid #E9E9E9;">
        <el-button type="primary" @click="save"><%=rb.getString("QueDing")%></el-button>
        <el-button @click="closeSlide"><%=rb.getString("QuXiao")%></el-button>
    </div>
	<el-dialog title="<%=rb.getString("TianJia")%>" id="addCellDialog" :visible.sync="addCellDialogShow" @close='closeAddCellDialog' :append-to-body="true" width="700" top="10vh">
      <div style="height:380px;">
		  	<!--:data="addCellTableData" :url="addCellTableUrl" -->
			<el-ctable	ref="addCellTable" :url="addCellTableUrl" :query-params="queryCellParams" :row-key="'serial_number'"
				:height="'100%'" pagination="true" rownumber="true" @load-success="tableLoadSuccess">
								
				<!-- 模糊查询 -- 软件升级 -->
				<template slot="toolbar">
					<div class="headQueryBox">
						<span style="padding-left: 10px;">eNB List</span>
						<el-query type="normal" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>" @query="queryCell"></el-query>
					</div>
				</template>
				<el-table-column label='' width="35" prop="">
					<template slot-scope="scope">
						<el-radio v-model="activeSerialNumber" :label="scope.row.serialNumber" @change="activeSerialNumberChange(scope.row)" >&nbsp;</el-radio>
					</template>
				</el-table-column>
				<el-table-column prop="connection_status" width="50">
					<template slot-scope="scope">
						<div :class="{
							'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
							'':scope.row.have_connected==2,
							'conn_exc':scope.row.connection_status=='Exception',
							'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serialNumber"></el-table-column>
				<el-table-column label="<%=rb.getString("HostName")%>" prop="cellName"></el-table-column>
				<el-table-column label="" prop="form" width="100">
					<template slot="header" slot-scope="scope">
						Halo <span class="haloXIconBox">X</span>
					</template>
					<template slot-scope="scope">
						HaloD<span class="haloDModeCodeCls">{{scope.row.form}}</span>
					</template>
				</el-table-column>
			</el-ctable>
            
		</div>
        <div class="addErrorCls" v-show="!activeSerialNumber"><%=rb.getString("ZhiShaoXuanZeYiGeSheBei")%></div>
        <div slot="footer" class="dialogFooterCls">
            <el-button type="primary" @click="addCellSubmit">OK</el-button>
            <el-button @click="closeAddCellDialog">Cancel</el-button>
        </div>
    </el-dialog>
</div>

<script>
	var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
    var specialTabVue = new Vue({
        el: '#special_tab',
        data() {
            var vm = this,
                validServerIP = function(rule,value,cb) {
                    if(value && !isIPv4(value)) {
                        cb('<%=rb.getString("IPDiZhi")%>');
                    }else if(value){
                        cb();
                    }else {
                        if(vm.form.enbL2TunnelEnable == 'true') {
                            cb('Required');
                        }else {
                            cb();
                        }
                    }
                },
				validateRange= function(rule,value,callback) {
					var min = rule.min;
					var max = rule.max;
					var mag = rule.mag;
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(vm.showHaloD){
						if(vm.halodMode === '0'){
							if(value == '' || value == undefined || value == null){
								callback(new Error(mag))
							}else{
								if(reg.test(value) && value >= min && value <= max){
									callback();
								}else{
									callback(new Error(mag))
								}
							}
						}else{
							callback();
						}
					}else{
						callback();
					}
                };

            return {
                mmeType: '',
                mmeCls: '',
                showMME: false,
                mmeVal: '',
                mme_plmn: '',
                mmeGroup: [],
                plmnGroup: [],

                serialNumber:'',
                policyName: '',
				mmeIP:'',
				mmeIpErr:false,
                form: {
                    lbtModel: '',
                    lbtTrigger: '',
                    lbtRetryPeriod: '',
                    lbtTime: '',
                    lbtPeriodSearch1: '',
                    lbtPeriodSearch2: '',

                    halobPeriod: '',
                    halobSwitch: '',
                    halobModel: '',
                   
                    sctpPort:'',
					mmePort:'',
                    mmeIp:'',
                },
                rules: {
                    enbL2ServerIp: [{validator: validServerIP}],
					sctpPort:[{validator:validateRange,min:1024,max:65535,mag:'<%=rb.getString("FanWei")%>：1024~65535,Integer'}],
					mmePort:[{validator:validateRange,min:1024,max:65535,mag:'<%=rb.getString("FanWei")%>：1024~65535,Integer'}],
                },
                casts: {
                    '3C628FFE94C4744C4CE76E7386CEFCDF': 'lbtModel',
                    '610CB3C37A82C0A9D2FE68B43CC0B376': 'lbtTrigger',
                    '6E368B0042A68A6C08EFC47CF63C9915': 'lbtRetryPeriod',
                    'BD10D631AC40B2A6AFCB6758A2C76DAF': 'lbtTime',
                    'FF2840D68CBDCFDC2E582EBE29A51ED0': 'lbtPeriodSearch1',
                    '76C0125C4419BB39F6010D2A87864EBB': 'lbtPeriodSearch2',
                    // halob Intel
                    '52A335BB16F15304EE1C00AF8828C908': 'halobPeriod',
                    'AC83E2E4348B9D7AA7A9A6332DA635CA': 'halobSwitch',
                    'D667DF1B0D547ECC7E3D46ABBC7D834E': 'halobModel',
                    // Intel_CR
                    'CFA5C3FCFEB8091E637AB8EDACBB044E': 'halobSwitch',
                    '17F145113E518CF4EFE090C98D826AEE': 'halobModel',
                    // MLN
                    '7DC9251CFE5A07F48F21105EF57B9A31': 'halobSwitch',
                    '8F581A6FBD69193FC5B6016D6DEDDF53': 'halobModel',
                    // Nova430
                    '74488A92A8FA42EAB068B0625ABF7087': 'halobSwitch',
                    '74488A92A8FA42EAB068B0625ABF7088': 'halobModel',
                    // Nova430i
                    '74488A92A8FA42EAB068B0625ABF6037': 'halobSwitch',
                    '74488A92A8FA42EAB068B0625ABF6038': 'halobModel',
                    // QA_436Q
                    '73ED21EFA1E1054A54F33F6FE2B15401': 'halobSwitch',
                    'F5B323783D32A2D3E3D3EFCE20932054': 'halobModel',

                    // HaloD 
                    'EE14A26C969A347055F1BDA597AA9A32': 'sctpPort',
                    'A6947BCC256A4D0C25FFA4DEDE00239E': 'mmePort',
                    '9989677C026BF3AFDD0A968980A17CD7': 'mmeIp',
                    '0F0B4079A1D9E3155DB83D30E9D0E302': 'sctpPort',
                    'D49B370E103EDDA52930D7649EE48CBA': 'mmePort',
                    'FE4D65F2D7304274097B0EFD41BB6F6A': 'mmeIp',
                    'B14F240D7853BA7FBE897637B53E8424': 'mmeIp',

                    'A1CFED2AAC45A5F9D9A6805D472F15E1': 'sctpPort',
                    '72DC89171F2BF3832CAAD1F51895CA3A': 'mmeIp',
                    '7BDEEF3FBC84630ADFEF7C8B1B8431DD': 'mmePort',

                    //mln
                    'A421EE574EA27F1ECF1356B1B8539163': 'sctpPort',
                    'B9ADC25C4C90D8F3F2205A4EA6066398': 'mmePort',
                    'A7B281ADD1E5FC8F761AA4F03361DEEF': 'mmeIp',
                },
               
                codeList: [],

                halodMode:'0',
				lockStatus:'0',
                
				activeLockType:false,
                mmeIpList:[
                    // '1.1.1.1','172.168.10.17'
                ],
                cellList:[
                    // {serialNumber:'1215000009214CS0024',form:'0',cellName:'Anhui-Telecom-longlonglonglong',enbId:'158',cellId:'68',ip:'192.168.172.161',lockStatus:'0'},
                    // {serialNumber:'1215000009214CS0025',form:'1',cellName:'Anhui-Telecom',enbId:'155',cellId:'67',ip:'192.168.1.6',lockStatus:'0'},
                    // {serialNumber:'1215000009214CS0026',form:'2',cellName:'Anhui-Telecom',enbId:'151',cellId:'66',ip:'192.168.1.8',lockStatus:'0'},
                ],
                defaultCellList:[],
				addCellDialogShow:false,
				queryCellParams:{
                    searchText:'',
                    serialNumber:'',
                },
				activeSerialNumber:'',
                activeCellRow:'',
                addCellTableUrl:'',
				addCellTableData:[
					{serialNumber:'1202000240194DP0027',cellName:'name_null',form:'1',enbId:'158',cellId:'68',ip:'192.168.172.161',lockStatus:'0'},
					{serialNumber:'1202000240194DP0028',cellName:'name_null',form:'2',enbId:'158',cellId:'68',ip:'192.168.172.161',lockStatus:'0'}
				],
				firstLoadTable:true
            }
        },
        computed: {
            showHalob() {
                return isSupportHalob == 'true';
            },
            showHaloD(){
                return this.halodMode == '0' || this.halodMode == '1' || this.halodMode == '2' ? true : false;
            },
			lockDis(){
				var masterArr=[]
				this.cellList.map((item)=>{
					if(item.form == '0'){
						masterArr.push(item)
					}
				})
				return masterArr.length !== 1 ? true : false;
			}
        },
        watch:{
            'form.mmeIp': function(val){
                var vm = this;
                vm.mmeIpList = val ? val.split(',') : [];
            },
			mmeGroup:function(){
				if(this.showMME == false){
					this.form.mmeIp = this.mmeGroup.join(',');
				}else{
					var str = '';
					this.mmeGroup.map(item=>{
						str += item.mme + ',' + item.plmn + ';'
					});
					this.form.mmeIp = str;
				}
			},
        },
        methods: {
            init() {
                var vm = this;
                
                vm.halodMode = settingVue.selectedRow.form;
                vm.showMME = ['CR-B4860','CR-B4860-DC','CR-B4860-SC','CR-B4860-CA','MLN','MLN-CA','MLN-SC','MLN-DC'].includes(settingVue.selectedRow.product);
                vm.serialNumber =  settingVue.selectedRow.serial_number;
                vm.queryCellParams.serialNumber = vm.serialNumber;
                if(vm.halodMode !== '3'){
                  vm.getHalodCellList();
                }
                vm.getParamNode();

                $('.form-operations').css({display:'none'});
            },

			isMMEExisted(type) {
				var vm = this,
					mme = vm.mmeVal,
					plmn = vm.mme_plmn,
					existed = false;

				vm.mmeGroup.map(function(item){
					if(type == 'mme'){
						if(item == mme) existed = true;
					}else {
						if(item.mme == mme && item.plmn == plmn) existed = true;
					}
				})

				return existed;
			},
			addMME(type){
				var value = this.mmeVal;
				this.mmeType = type;

            	if(!isValidIP(value) || this.isMMEExisted(type)){
					this.mmeCls = 'is-error';
				}else{
					this.mmeCls = '';
					if(type == 'mme'){
						this.mmeGroup.push(this.mmeVal);
					}else{
						this.mmeGroup.push({mme:this.mmeVal,plmn:this.mme_plmn})
					}
					this.mmeVal = '';
				}
			},
			removeMME(index){
				this.mmeVal = '';
				this.mme_plmn = '';
				this.mmeGroup.splice(index,1);
			},

            getHalodCellList(){ // 查询HaloD 关系表
                var vm =this,
                    urls = '${ctx}/cell/halod/queryHalodRelationInfo4Datagrid.action',
                     params = {
                        serialNumber:settingVue.selectedRow.serial_number,
                        page:1,
                        rows:20
                    };
                axios.post(urls,stringify(params)).then(function(response){
                    var data = response.data
                    vm.cellList = data.rows;
                    vm.defaultCellList = JSON.parse(JSON.stringify(data.rows));
                    vm.lockStatus = JSON.parse(JSON.stringify(vm.cellList[0].lockStatus));
                })
            },
            getParamNode() {
                var vm = this,
                    codes = [],
                    url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                    params = {
                        id: '44',
                        smallCellCode: settingVue.selectedRow.small_cell_code
                    };

                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data;
                    
                    if(data && Array.isArray(data)) {
                        data.map(function(item){
                            item.groups.map(function(group){
                                group.list.map(function(m){
                                    codes.push(m.name);
                                    // 执行赋值
                                    vm.setValue(m);
                                    /*
                                    if(m.label == 'MME IP'){
                                    	vm.showMME = m.type == 'bind' ? true : false;
                                    	if(vm.showMME) {
                                    		m.value.split(';').map(function(mmePlmn){
                                    			var mmeArr = mmePlmn.split(',');
                                    			
                                    			vm.mmeGroup.push({mme: mmeArr[0], plmn: mmeArr[1]})
                                    		});
                                            
											// init plmnGroup
											if(m.bindObj && m.bindObj.data) {
												m.bindObj.data.map(function(plmn){
													vm.plmnGroup.push(plmn);
												});
											}
                                    	}else {
											m.value.split(',').map(function(mmeIp){
												vm.mmeGroup.push(mmeIp)
                                    		});
                                    	}
                                    }
                                    */
                                });
                            });
                        });

                        vm.$nextTick(function(){
                            initForm(vm.$refs.form);
                        })

                        vm.codeList = codes;
                    }
                });
            },
            setValue(item) {
                var vm = this,
                    code = item.name,
                    value = item.value;

                var key = vm.casts[code];

                vm.form[key] = value;
            },
            hasKey(key) {
                var vm = this,
                    has = false;

                vm.codeList.map(function(name){
                    if(vm.casts[name] == key) has = true;
                });

                return has;
            },
			// haloD mmeIP 新增
			addMmeIP(){
				var vm = this;
				if(vm.mmeIP){
					if(regIp.test(vm.mmeIP)){
                        var isExist =  vm.mmeIpList.some(item =>item == vm.mmeIP);
						if(isExist){
                            vm.$message.error('<%=rb.getString("YiCunZai")%>')
                            return 
						}
						vm.mmeIpErr = false;
                        var arr = [];
                        arr.push(vm.mmeIP);
                        var newMmeIpList = [...vm.mmeIpList,...arr]
						vm.form.mmeIp = newMmeIpList.join(',');
					}else{
						vm.mmeIpErr = true;
					}
				}else{
					vm.mmeIpErr = true;
				}
			},
			// 关系锁定事件
			lockTypeChange(){
				var vm = this;

				vm.activeLockType = !vm.activeLockType;
                if(vm.lockStatus == '0'){
                    vm.lockStatus = '1'
                }else{
                    vm.lockStatus = '0'
                }
			},
			// HaloD 新增cell事件 打开可新增cell表格弹窗
			addCellClick(){
				var vm = this;
				vm.addCellDialogShow = true;
                vm.addCellTableUrl='${ctx}/cell/halod/queryHalodDeviceInfo4Datagrid.action?rd='+Math.random().toString();
			},
			// 可新增cell表格 模糊搜索
			queryCell(val){
				var vm = this;
				vm.queryCellParams.searchText = val;
			},
			// cell选中改变事件
			activeSerialNumberChange(row){
                var vm = this;

                vm.activeCellRow = row;
			},
			// 新增cell提交事件
			addCellSubmit(){
				var vm = this;

                var isExist =  vm.cellList.some(item =>item.serialNumber == vm.activeSerialNumber);
                if(isExist){
                    vm.$message.error('<%=rb.getString("YiCunZai")%>')
                    return 
                }
                vm.cellList.push(vm.activeCellRow);
                vm.addCellDialogShow = false;
			},
            // cell 删除
            cellItemDel(item){
                var vm = this;
                vm.cellList = vm.cellList.filter((items)=>{
					return items.serialNumber != item.serialNumber
				})
            },
			// 关闭新增cell弹窗
			closeAddCellDialog(){
				var vm = this;
                vm.addCellDialogShow = false;
				vm.firstLoadTable = true;
                setTimeout(()=>{
                    vm.activeSerialNumber = '';
                    vm.activeCellRow = '';
                },200);
                
			},
			// 可新增cell表格 第一次加载
			tableLoadSuccess(data){
				var vm = this;
				if(vm.firstLoadTable){
					if(data.rows && data.rows.length > 0){
						vm.activeSerialNumber = data.rows[0].serialNumber;
                        vm.activeCellRow = data.rows[0];
						vm.firstLoadTable = false;
					}
				}
				
			},
            // mme ip删除事件
            delHalodMmeIp(val){
                var vm = this,
                    newMmeIpList = [];
                vm.mmeIpList.map((item)=>{
                    if(item !== val){
                        newMmeIpList.push(item)
                    }
                })
                vm.form.mmeIp = newMmeIpList.join(',');
            },
            // 自动探测/更新
            refrshCellList(){
                var vm =this,
                    urls = '${ctx}/cell/halod/queryHalodRelationInfo4Datagrid.action',
                     params = {
                        serialNumber:settingVue.selectedRow.serial_number,
                        page:1,
                        rows:20
                    };
                axios.post(urls,stringify(params)).then(function(response){
                    var data = response.data;
                    vm.cellList = data.rows;
                    vm.defaultCellList = JSON.parse(JSON.stringify(data.rows));
                })
            },
            isNull(val){
                if(val==undefined || val == null || val =="") return true;
                else return false;
            },
            getNameByProp(prop) {
                var vm = this,
                    reg = /^\w*\.\d*\.\w*$/,
                    key = prop;
                
                if(reg.test(prop)) {
                    
                }else {
                    vm.codeList.map(function(name){
                        if(vm.casts[name] == prop) {
                            key = name;
                        }
                    });
                }

                return key;
            },
            save() {
                var vm = this;

                var subParams = {},
                    isChanged = isFormChanged(vm.$refs.form),
                    isCellListChange = JSON.stringify(vm.defaultCellList) == JSON.stringify(vm.cellList) ? false : true;

                if(!isChanged && !isCellListChange && !vm.activeLockType) {
                    showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
                    return;
                }

                vm.$refs.form.fields.map(function(field){
                    var key = vm.getNameByProp(field.prop);

                    if(Array.isArray(field.fieldValue)){
                        var vList = field.fieldValue.map(function(item){return item}),
                            oList = (field.reinitialValue||[]).map(function(item){return item}),
                            val = vList.sort().join(','),
                            orVal = oList.sort().join(',');

                        if(val != orVal) {
                            subParams[key] = val;
                        };
                    }else{
                        if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
                            
                        }else if(field.fieldValue != field.reinitialValue) {
                                subParams[key] = field.fieldValue;
                        };
                    }
                });
                
                vm.$refs.form.validate(function(valid){
                    if(valid) {
                        $('#setting_form_cnt').addClass('loading');
                        var rowCode = settingVue.selectedRow.small_cell_code,
                            url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
                        if(isChanged && (isCellListChange || vm.activeLockType)) {
                            $.post(url, {params: JSON.stringify(subParams)}, function(data){
                                if(data.success){
                                    if(vm.halodMode !== '3'){
                                        var urls = '',
                                            subLockStatus = vm.lockStatus,
                                            masterSerialNumberList=[],
                                            snList=[],
                                            paramsData={
                                                operatorCode:operatorCodeGloab,
                                                masterSerialNumber:''
                                            };
                                            subCellList = JSON.parse(JSON.stringify(vm.cellList));
                                        subCellList.map((item,index)=>{
                                           snList.push(item.serialNumber);
                                           if(item.form == '0'){
                                               masterSerialNumberList.push(item.serialNumber)
                                           }
                                        })
                                        paramsData.masterSerialNumber = masterSerialNumberList.join(',');
                                        
                                        if(subLockStatus == '1'){
                                            urls = '${ctx}/cell/halod/saveHaloDRelation.action';
                                            paramsData.slaveSerialNumber = snList.join(',');
                                        }else{
                                            urls = '${ctx}/cell/halod/releaseHaloDRelation.action';
                                        }
                                        axios.post(urls,stringify(paramsData)).then(function(response){
                                            var data = response.data;
                                            var message = '<%=rb.getString("ChengGong")%>';
                                            if(data["success"]){
                                                vm.$message({
                                                    message:message,
                                                    type:'success',
                                                })
                                                closeSettingPanel();
                                                if(Render.validResult.reboot) {
                                                    $.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
                                                        if (!data["success"]) {
                                                            showMsg('error_msg',data["message"]);
                                                        }
                                                    }, "json");
                                                }
                                                addEdit = true;
                                            }else{
                                                vm.$message.error(data["message"])
                                            }
                                        })  
                                        
                                    }else{
                                        showMsg("success_msg",'<%=rb.getString("ChengGong")%>')
                                        closeSettingPanel();
                                        if(Render.validResult.reboot) {
                                            $.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
                                                if (!data["success"]) {
                                                    showMsg('error_msg',data["message"]);
                                                }
                                            }, "json");
                                        }
                                        addEdit = true;
                                    }
                                }else{
                                    if(data.validMsg){
                                        $.messager.alert(TISHI,data.validMsg,'warning');
                                    }else{
                                        $.messager.confirm(TISHI,'<%=rb.getString("JiZhanSheZhiShiBai")%>',function(r){
                                            if(r) closeSettingPanel();;
                                        });
                                    }
                                    addEdit = false
                                }
                                
                                Render.validResult.reboot = false;
                                $('#setting_form_cnt').removeClass('loading')
                            },'json');
                        }else if(isChanged && !isCellListChange && !vm.activeLockType){
                            $.post(url, {params: JSON.stringify(subParams)}, function(data){
                                if(data.success){
                                    showMsg("success_msg",'<%=rb.getString("ChengGong")%>')
                                    closeSettingPanel();
                                    if(Render.validResult.reboot) {
                                        $.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
                                            if (!data["success"]) {
                                                showMsg('error_msg',data["message"]);
                                            }
                                        }, "json");
                                    }
                                    addEdit = true;
                                }else{
                                    if(data.validMsg){
                                        $.messager.alert(TISHI,data.validMsg,'warning');
                                    }else{
                                        $.messager.confirm(TISHI,'<%=rb.getString("JiZhanSheZhiShiBai")%>',function(r){
                                            if(r) closeSettingPanel();;
                                        });
                                    }
                                    addEdit = false
                                }
                                
                                Render.validResult.reboot = false;
                                $('#setting_form_cnt').removeClass('loading')
                            },'json');
                        }else if((!isChanged && isCellListChange) || (!isChanged && vm.activeLockType)) {
                           
                            var urls = '',
                                subLockStatus = vm.lockStatus,
                                masterSerialNumberList=[],
                                snList=[],
                                paramsData={
                                    operatorCode:operatorCodeGloab,
                                    masterSerialNumber:''
                                };
                                subCellList = JSON.parse(JSON.stringify(vm.cellList));
                            subCellList.map((item,index)=>{
                                snList.push(item.serialNumber);
                                if(item.form == '0'){
                                    masterSerialNumberList.push(item.serialNumber)
                                }
                            })
                            paramsData.masterSerialNumber = masterSerialNumberList.join(',');
                           
                            if(subLockStatus == '1'){
                                urls = '${ctx}/cell/halod/saveHaloDRelation.action';
                                paramsData.slaveSerialNumber = snList.join(',');
                            }else{
                                urls = '${ctx}/cell/halod/releaseHaloDRelation.action';
                            }
                            axios.post(urls,stringify(paramsData)).then(function(response){
                                var data = response.data;
                                var message = '<%=rb.getString("ChengGong")%>';
                                if(data["success"]){
                                    vm.$message({
                                        message:message,
                                        type:'success',
                                    })
                                    closeSettingPanel();
                                }else{
                                    vm.$message.error(data["message"])
                                }
                                $('#setting_form_cnt').removeClass('loading')
                            })  
                        }
                    }
                });
            },
            closeSlide() {
                closeSettingPanel();
            }
        },
        mounted() {
            this.init();
        }
    });
</script>