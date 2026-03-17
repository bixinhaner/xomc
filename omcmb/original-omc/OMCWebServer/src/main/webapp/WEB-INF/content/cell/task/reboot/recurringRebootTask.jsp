<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#recurringRebootTask .el-form-item__label{
	line-height:26px;
}
#recurringRebootTask .modeItem .el-radio{
	display:block;
	margin-left:0px;
	margin-bottom:20px;
} 
#recurringRebootTask .modeItem{
	margin-top:20px;
}
#recurringRebootTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#recurringRebootTask .el-radio{
	width: 200px;
}
#recurringRebootTask .runTimeSettingBox .el-radio{
	width: 120px;
}
#recurringRebootTask .el-form-item__error{
	white-space: nowrap;
}
#recurringRebootTask .el-form-item__error{
	margin-top: 0px;
}
#recurringRebootTask .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#recurringRebootTask .titleStyML{
	margin-left: 20px;
}
#recurringRebootTask .runTimeSettingBox{
	padding: 20px 40px;
}
#recurringRebootTask .dataBoxCls{
	display: flex;
	padding-top: 20px;
}
#recurringRebootTask .dataBoxCls .el-form-item{
	width: 300px;
	margin-bottom: 0px;
}
#recurringRebootTask .dataBoxCls .el-input__icon::before{
	color: #4D84FF;
}
#recurringRebootTask .itemCls{
	margin-left:45px;
	margin-top:10px;
	margin-bottom: 20px;
}
#recurringRebootTask .runDurationBox{
	height: 28px;
	margin-left: 20px;
	box-sizing: border-box;
	display: inline-block;
}
#recurringRebootTask .itemDisplay .el-form-item__content{
	display: flex;
}
#recurringRebootTask .runDurationEndNameCls{
	height: 28px;
	box-sizing: border-box;
	display: inline-block;
	line-height: 28px;
	padding: 0px 10px;
	background-color: #F5F7FA;
	border:1px solid #E9E9E9;
	border-left: none;
	border-radius: 0px 2px 2px 0px;
}
#recurringRebootTask .runTimeSettingBox .el-date-editor.el-input{
	width: 130px !important;
}
#recurringRebootTask .runTimeSettingBox .el-input__inner{
	width: 130px !important;
}
#recurringRebootTask .runTimeSettingBox .el-input__icon{
	line-height: 28px;
}
#recurringRebootTask .noteCls{
	font-size:12px;
	color:#999999;
	display:inline-block;
	margin-left: 10px;
}
#recurringRebootTask .specifiedTypeBox{
	flex: 1;
	padding-top: 30px;
	padding-left: 40px;
}
#recurringRebootTask .specifiedTypeBox .el-radio__label{
	font-size: 12px !important;
}
#recurringRebootTask .specifiedTypeBox .el-radio+.el-radio{
	margin-left: 0px;
}
#recurringRebootTask .deviceSpecifiedBox{
	width: 240px;
	height: 370px;
	box-sizing: border-box;
	border:1px solid #E9E9E9;
	border-right: none;
}
#recurringRebootTask .deviceSpecifiedTitle{
	height: 36px;
	width: 240px;
	font-size: 12px;
	line-height: 36px;
	color:#333333;
	font-weight: bold;
	text-align: center;
	background: #F6F7FB;
	box-sizing: border-box;
	border-bottom:1px solid #E9E9E9;
}
#recurringRebootTask .deviceTableBox{
	width: calc(100% - 200px)!important;
	font-size: 12px !important;
	margin:auto;
	display:unset;
}
#recurringRebootTask .deviceTableBox .pairgrid-right{
	top:40px!important;
	height: calc(100% - 40px)!important;
}
#recurringRebootTask .deviceTableBox .el-pairgrid-title{
	top:15px!important;
	right: 15px!important;
}
#recurringRebootTask .deviceTableBox .transition-box .el-form-item{
	display: inline-block;
	margin-right: 30px;
}
#recurringRebootTask .specifiedTypeBox .el-radio{
	margin-bottom: 20px;
	display: block;
}
#recurringRebootTask .noLabel .el-form-item__content {
	margin-left:0 !important;
}
</style>

<%-- 配置定时重启任务 --%>
<div id="recurringRebootTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" style='padding-top:20px;' label-position="left"  :hide-required-asterisk=true>
		<div class="group-title not-extend titleStyML" >
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheZhiChongQiTiaoJian")%></span>
		</div>
		<el-form-item label='<%=rb.getString("YunXingShiChangXianZhi")%>' :label-width="dynLabelWidth" prop='duration' class="itemCls itemDisplay">
            <el-radio-group :disabled="viewFlag" v-model="ruleForm.runDuraLimit" style='margin-top:7px;' @change="runDuraLimitChange">
                <el-radio label="0"><%=rb.getString("RenYiYunXingShiChang")%></el-radio>
                <el-radio label="1" style="width:unset"><%=rb.getString("YunXingChaoGuo")%></el-radio>
            </el-radio-group>
			<div class="runDurationBox">
				<el-form-item label-width="0px" prop='runDuration' >
					<el-input maxlength=100 :disabled="viewFlag || ruleForm.runDuraLimit == '0'" v-model="ruleForm.runDuration" size="mini" style="width:60px;height:28px;line-height:28px;"></el-input>
					<div class="runDurationEndNameCls"><%=rb.getString("XiaoShi")%></div>
				</el-form-item>
			</div>
		</el-form-item>

		<el-form-item label='' :label-width="dynLabelWidth" prop='deviceAssign' class="itemCls noLabel" style="margin-right:0px;margin-bottom:0px;width:1150px;">
			<div style="display: flex;">
				<div class="deviceSpecifiedBox">
					<div class="deviceSpecifiedTitle"><%=rb.getString("SheBeiLieBiaoBiaoTi")%></div>
					<div class="specifiedTypeBox">
						<el-radio-group :disabled="viewFlag" v-model="ruleForm.deviceAssign" style='margin-top:7px;'>
							<el-radio label="0"><%=rb.getString("QuanBu")%></el-radio>
							<el-radio label="1"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
							<el-radio label="2"><%=rb.getString("ZhiDingBuZhiXing")%></el-radio>
						</el-radio-group>
					</div>
				</div>
				<div class="deviceTableBox">
					<el-pairgrid 
						id="select_device_list" 
						v-if="ruleForm.deviceAssign !== '0'" 
						:readonly='viewFlag' 
						:rownumber="true" 
						ref="devicePairgrid" 
						@checked-change='deviceSelectChange'  
						:right-url="deviceRightUrl" 
						:left-url="deviceLeftUrl" 
						:height="height" 
						row-key="smallCellCode" 
						:query-params="deviceQueryParams" 
						:title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}"
					 >
						<template slot="left">
							<el-table-column type='selection' width="45"></el-table-column>
							<el-table-column prop="connection_status" width="50">
								<template slot-scope="scope">
									<div :class="{
										'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
										'':scope.row.have_connected==2,
										'conn_exc':scope.row.connection_status=='Exception',
										'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
								</template>
							</el-table-column>
							<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"></el-table-column>
							<el-table-column prop="hostName" label='<%=rb.getString("HostName") %>' min-width="110" show-overflow-tooltip ></el-table-column>
						</template>
						<template slot='toolbar'>
							<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>"></el-query>
						</template>
						<template slot='right'>
							<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
						</template>
					</el-pairgrid>
					<el-ctable 
						id="all_device_list" 
						v-if="ruleForm.deviceAssign == '0'" 
						:query-params="deviceQueryParams" 
						style="border:1px solid #E9E9E9;" 
						ref="all_device_list"
						:url="deviceLeftUrl" 
						:height="height" 
						pagination="true" 
						>
						<template slot='toolbar'>
							<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>"></el-query>
						</template>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"></el-table-column>
						<el-table-column prop="hostName" label='<%=rb.getString("HostName") %>' min-width="110" show-overflow-tooltip ></el-table-column>
					</el-ctable>
				</div>
			</div>
		</el-form-item>
		<el-form-item prop='selectedENB' style="margin:0px 0px 30px 45px;" :label-width="dynLabelWidth">
			<el-input v-model='ruleForm.selectedENB' v-show="false"></el-input>
		</el-form-item>
		
		<el-form-item label='' :label-width="dynLabelWidth" prop='versionAssign' class="itemCls noLabel" style="margin-bottom:0px;margin-right:0px;width:1150px;">
			<div style="display: flex;">
				<div class="deviceSpecifiedBox">
					<div class="deviceSpecifiedTitle"><%=rb.getString("BanBenLieBiao")%></div>
					<div class="specifiedTypeBox">
						<el-radio-group :disabled="viewFlag" v-model="ruleForm.versionAssign" style='margin-top:7px;'>
							<el-radio label="0"><%=rb.getString("QuanBu")%></el-radio>
							<el-radio label="1"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
							<el-radio label="2"><%=rb.getString("ZhiDingBuZhiXing")%></el-radio>
						</el-radio-group>
					</div>
				</div>
				<div class="deviceTableBox">
					<el-pairgrid  
						id="select_version_list" 
						v-if="ruleForm.versionAssign !== '0'" 
						:readonly='viewFlag' 
						:rownumber="true" 
						ref="versionPairgrid" 
						@checked-change='versionSelectChange' 
						:right-url="versionRightUrl" 
						:left-url="versionLeftUrl" 
						:height="height" 
						row-key="software_version" 
						:query-params="versionQueryParams" 
						:title="versionTitle" 
						:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}"
					 >
						<template slot="left">
							<el-table-column type='selection' width="45"></el-table-column>
							<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>' min-width="200"></el-table-column>
						</template>
						<template slot='toolbar'>
							<el-query type="normal" @query="queryVersionList" placeholder="<%=rb.getString("BanBen")%>"></el-query>
						</template>
						<template slot='right'>
							<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>'></el-table-column>
						</template>
					</el-pairgrid>
					<el-ctable 
						id="all_version_list" 
						v-if="ruleForm.versionAssign == '0'" 
						:query-params="versionQueryParams" 
						style="border:1px solid #E9E9E9;" 
						ref="all_version_list"
						:url="versionLeftUrl" 
						:height="height" 
						pagination="true" 
						>
						<template slot='toolbar'>
							<el-query type="normal" @query="queryVersionList" placeholder="<%=rb.getString("BanBen")%>"></el-query>
						</template>
						<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>' min-width="200"></el-table-column>
					</el-ctable>
				</div>
			</div>
		</el-form-item>
		<el-form-item prop='selectedVersion' style="margin:0px 0px 30px 45px;" :label-width="dynLabelWidth">
			<el-input v-model='ruleForm.selectedVersion' v-show="false"></el-input>
		</el-form-item>
		
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("YunXingShiJianSheZhi")%></span>
		</div>
		<div class="runTimeSettingBox">
            <el-radio-group :disabled="viewFlag" v-model="ruleForm.onlyOnce" style='margin-top:7px;' @change="onlyOnceChange">
                <el-radio label="0"><%=rb.getString("DanCiRenWu")%></el-radio>
                <el-radio label="1"><%=rb.getString("ZhouQiRenWu")%></el-radio>
            </el-radio-group>
            <div class="dataBoxCls">
                <el-form-item prop='startDay' label='<%=rb.getString("KaiShiRiQi")%>' label-width="120px">
                    <el-date-picker 
						value-format="yyyy-MM-dd" 
						v-model='ruleForm.startDay' 
						type="date"
						:disabled="viewFlag"
						@change="startDataChange"
					></el-date-picker>	
                </el-form-item>
                 <el-form-item prop='endDay' label='<%=rb.getString("JieShuRiQi")%>' label-width="120px">
                    <el-date-picker 
						:disabled="viewFlag || ruleForm.onlyOnce == '0'" 
						value-format="yyyy-MM-dd" 
						v-model='ruleForm.endDay' 
						type="date"
						@change="endDataChange"
					></el-date-picker>	
                </el-form-item>
            </div>
			<div class="dataBoxCls">
                 <el-form-item prop='startTime' label='<%=rb.getString("KaiShiShiJian")%>' label-width="120px">
                    <el-time-picker  
						v-model='ruleForm.startTime' 
						:disabled="viewFlag" 
						value-format="HH:mm:ss"
						 @change="startTimeChange"
						 @focus="defaultStartTimes"
					></el-date-picker>	
                </el-form-item>
                 <el-form-item prop='endTime' label='<%=rb.getString("JieShuShiJian")%>' label-width="120px">
                    <el-time-picker  
						v-model='ruleForm.endTime' 
						:disabled="viewFlag" 
						value-format="HH:mm:ss" 
						@change="endTimeChange"
						 @focus="defaultEndTimes"
					></el-date-picker>	
                </el-form-item>
            </div>
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ChongQiShuLiangXianZhi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("ChongQiJiZhanZuiDaBingFaShuLiang")%>' prop='rebootLimitNum' :label-width="rebootLimitNumLabelWidth" class="itemCls" style="margin-top:20px;" >
			<el-input v-model="ruleForm.rebootLimitNum" :disabled="viewFlag" size="mini" style="width:100px;height:28px;line-height:28px;"></el-input>
			<div class="noteCls"><%=rb.getString("FanWei")%>： 1-500</div>
		</el-form-item>
		<div class="itemCls" style="display:flex;align-items: center;">
			<el-checkbox :disabled="viewFlag" v-model="maxNumSwitch" true-label="1" false-label="0" @change="rebootMaxNumCheckChange"></el-checkbox>
			<el-form-item label='<%=rb.getString("MeiGeZhouQiChongQiZuiDaShuLiang")%>' prop='rebootMaxNum' :label-width="rebootMaxNumLabelWidth" style="margin-bottom:0px;width:800px;margin-left:10px;">
				<el-input :disabled="viewFlag || maxNumSwitch == '0'" v-model="ruleForm.rebootMaxNum" size="mini" style="width:100px;height:28px;line-height:28px;"></el-input>
				<div class="noteCls"><%=rb.getString("FanWei")%>：1-1000</div>
			</el-form-item>
		</div>
	</el-form>
</div>
<script type="text/javascript">
new Vue({
	el:'#recurringRebootTask',
	data(){
		var vm = this;
		var validateRunDuration = (rule,value,callback) => {
				var reg =/(^[1-9]\d*$)/;
				if(this.ruleForm.runDuraLimit == '0'){
					callback()
				}else{
					if(value == ""){
						callback(new Error('<%=rb.getString("ZhengXing")%>'))
					}else if(reg.test(value)){
						callback()
					}else{
						callback(new Error('<%=rb.getString("ZhengXing")%>'))
					}
				}
			    
		    },
		    validateStartDate = (rule,value,callback) => {
				var startDay,endDay;
				
				startDay = value + ' 00:00:00';
				endDay = this.ruleForm.endDay + ' 00:00:00';
				if(value){
					if(this.ruleForm.endDay){
						var validTimeResult = validateStartAndStopTime(startDay,endDay);
						if(validTimeResult == 'false'){
							callback(new Error('<%=rb.getString("ShiJianFanWeiBuZhengQue")%>'))
						}else{
							callback()
						}
					}else{
						callback()
					}
				}else{
					callback(new Error('<%=rb.getString("KaiShiShiJianBuNengKong")%>'))
				}
		    },
            validateEndDate = (rule,value,callback) => {
			    var startDay,endDay;
				
				startDay = this.ruleForm.startDay + ' 00:00:00';
				endDay = value + ' 00:00:00';
				if(value){
					if(this.ruleForm.startDay){
						var validTimeResult = validateStartAndStopTime(startDay,endDay);
						if(validTimeResult == 'false'){
							callback(new Error('<%=rb.getString("ShiJianFanWeiBuZhengQue")%>'))
						}else{
							callback()
						}
					}else{
						callback()
					}
				}else{
					callback()
				}
		    },
			validateStartTime = (rule,value,callback) => {
				var startTime,endTime;
				
				startTime = '2021-03-12 ' + value;
				endTime = '2021-03-12 ' + this.ruleForm.endTime;
				if(value){
					if(this.ruleForm.endTime){
						var validTimeResult = validateStartAndStopTime(startTime,endTime);
						if(validTimeResult == 'false'){
							callback(new Error('<%=rb.getString("ShiJianFanWeiBuZhengQue")%>'))
						}else{
							callback()
						}
					}else{
						callback()
					}
				}else{
					callback(new Error('<%=rb.getString("KaiShiShiJianBuNengKong")%>'))
				}
			},
			validateEndTime = (rule,value,callback) => {
				var startTime,endTime;
				
				startTime = '2021-03-12 ' + this.ruleForm.startTime;
				endTime = '2021-03-12 ' + value;
				if(value){
					if(this.ruleForm.startTime){
						var validTimeResult = validateStartAndStopTime(startTime,endTime);
						if(validTimeResult == 'false'){
							callback(new Error('<%=rb.getString("ShiJianFanWeiBuZhengQue")%>'))
						}else{
							callback()
						}
					}else{
						callback()
					}
				}else{
					callback()
				}
			},
			validateRebootLimitNum = (rule,value,callback) => {
				var reg =/^(500|[1-4][0-9][0-9]|[1-9][0-9]|[1-9])$/;
				
				if(value == ""){
					callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>:1-500'))
				}else if(reg.test(value)){
					callback()
				}else{
					callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>:1-500'))
				}
				
			},
			validateRebootMaxNum = (rule,value,callback) => {
				var reg =/^(\+?[1-9]\d{0,2}|\+?1000)$/;
				if(this.maxNumSwitch == '1'){
					if(value == ""){
						callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>:1-1000'))
					}else if(reg.test(value)){
						callback()
					}else{
						callback(new Error('<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>:1-1000'))
					}
				}else{
					callback()
				}
			},
			validateSelectedENB = (rule,value,callback) => {

				if(this.ruleForm.deviceAssign == '0'){
					callback()
				}else{
					if(value){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
					}
				}
			},
			validateSelectedVersion = (rule,value,callback) => {

				if(this.ruleForm.versionAssign == '0'){
					callback()
				}else{
					if(value){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuXuYaoTianJiaDeBanBen")%>'))
					}
				}
			};
		return {
			viewFlag:false,
			deviceLeftUrl:'${ctx}/task/recurringReboot/getEnbListPageData.action',
			deviceRightUrl:'',
			height:'370px',
			versionLeftUrl:'${ctx}/task/recurringReboot/getSoftwareVersionList.action',
			versionRightUrl:'',
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			versionTitle:['','<%=rb.getString("YiXuan")%>'],
			versionSelection: [],
			deviceSelection: [],
			ruleForm:{
				runDuraLimit:'0',
                runDuration:'',
				selectedENB:'',
				selectedVersion:'',
				deviceAssign:'1',  // 0 所有  1 指定执行  2 指定不执行
                versionAssign:'1', // 0 所有  1 指定执行  2 指定不执行
				startDay:'',
				endDay:'',
				startTime:'',
				endTime:'',
                onlyOnce:'0',  // 0 单次   1 周期
				rebootLimitNum:'', // 最大并发数
				rebootMaxNum:'',    // 最大重启数量
			},
			maxNumSwitch:'0',
			rules:{
				runDuration:[
					{validator:validateRunDuration,trigger:'blur'}
				],
				startDay:[
					{validator:validateStartDate,trigger:'change'}
				],
				endDay:[
					{validator:validateEndDate,trigger:'change'}
				],
				startTime:[
					{validator:validateStartTime,trigger:'change'}
				],
				endTime:[
					{validator:validateEndTime,trigger:'change'}
				],
				rebootLimitNum:[
					{validator:validateRebootLimitNum,trigger:'blur'},
				],
				rebootMaxNum:[
					{validator:validateRebootMaxNum,trigger:'blur'},
				],
				selectedENB:[
					{validator:validateSelectedENB,trigger:'change'},
				],
				selectedVersion:[
					{validator:validateSelectedVersion,trigger:'change'},
				],
			},
			deviceQueryParams:{
				searchText:'',
				timeZone:timeZone,
				operatorCode:operator_code
			},
			versionQueryParams:{
				searchText:'',
				timeZone:timeZone,
				operatorCode:operator_code
			},
			runDurationFlag:false,
		}
	},
	computed:{
		dynLabelWidth() {
			return isLocalZH == true ? '120px' : '160px';
		},
		rebootLimitNumLabelWidth(){
			return isLocalZH == true ? '160px' : '420px';
		},
		rebootMaxNumLabelWidth(){
			return isLocalZH == true ? '160px' : '290px';
		},
    },
	methods:{ 
		// 初始化
		init(type){
			var vm = this,
				params={
					operatorCode:operator_code,
					timeZone:timeZone
				};
			if(type == '1'){
				vm.viewFlag = true;
			}
			
			axios.post('${ctx}/task/recurringReboot/getConfigInfo.action',stringify(params)).then(function(response){
				var data = response.data;
				if(data.id){
					Object.assign(vm.ruleForm, data);
					if(vm.ruleForm.rebootMaxNum){
						vm.maxNumSwitch = '1';
					}else{
						vm.maxNumSwitch = '0';
					}
					if(data.deviceAssign != '0'){
						vm.deviceRightUrl = '${ctx}/task/recurringReboot/getSelectedEnbPageData.action?operatorCode='+operator_code;
					}
					if(data.versionAssign != '0'){
						vm.versionRightUrl = '${ctx}/task/recurringReboot/getSelectedSoftwareVersionList.action?operatorCode='+operator_code;
					}
					initForm(vm.$refs.ruleForm);
				}
			}).catch(function(error){})
		},
		// 默认开始时间
		defaultStartTimes(){
			var vm = this;

			if(!vm.ruleForm.startTime){
				vm.ruleForm.startTime = formatDate(Date.getNow()).slice(-8)
			}
		},
		// 默认结束时间
		defaultEndTimes(){
			var vm = this;

			if(!vm.ruleForm.endTime){
				vm.ruleForm.endTime = formatDate(Date.getNow()).slice(-8)
			}
		},
		// 运行时长限制切换事件 0 -- 任意  1 -- 固定
		runDuraLimitChange(val){
			var vm = this;

			if(val == '0'){
				vm.ruleForm.runDuration = '';
				vm.$refs.ruleForm.validateField('runDuration');
			}
		},
		// 任务类型切换事件  0 -- 单次  1 -- 周期
		onlyOnceChange(val){
			var vm = this;

			if(val == '0'){
				vm.ruleForm.endDay = '';
				vm.$refs.ruleForm.validateField('endDay');
			}
		},
		// 开始日期改变事件
		startDataChange(val){
			var vm = this;
			vm.$refs.ruleForm.validateField('endDay');
		},
		// 结束日期改变事件
		endDataChange(val){
			var vm = this;
			vm.$refs.ruleForm.validateField('startDay');
		},
		// 开始时间改变事件
		startTimeChange(val){
			var vm = this;
			vm.$refs.ruleForm.validateField('endTime');
		},
		// 结束时间改变事件
		endTimeChange(val){
			var vm = this;
			vm.$refs.ruleForm.validateField('startTime');
		},
		// 是否启用最大重启数事件
		rebootMaxNumCheckChange(val){
			var vm = this;

			if(val == '0'){
				vm.ruleForm.rebootMaxNum = '';
				vm.$refs.ruleForm.validateField('rebootMaxNum');
			}
		},
		// 设备列表 模糊查询
		queryDeviceList(val){
			var vm = this;
			vm.deviceQueryParams.searchText = val;
		},
        // 版本列表 模糊查询
		queryVersionList(val){
			var vm = this;
			vm.versionQueryParams.searchText = val;
		},
		// 设备选择事件
		deviceSelectChange(){
			
			var vm =this,
				data = vm.$refs.devicePairgrid.getData();
				selectedENB = '',selectedENBList = [];
			if(data.length != 0){
				data.map(function(item){
					selectedENBList.push(item.smallCellCode)
				})
			}
			selectedENB = selectedENBList.join(',');
			vm.ruleForm.selectedENB = selectedENB
		},
		// 版本选择事件
		versionSelectChange(){
	        var vm =this,
				data = vm.$refs.versionPairgrid.getData();
				selectedVersion = '',
				selectedVersionList=[];
			if(data.length != 0){
				data.map(function(item){
					selectedVersionList.push(item.software_version)
				})
			}
			selectedVersion = selectedVersionList.join(',');
			vm.ruleForm.selectedVersion = selectedVersion;
	    },
		
		// 提交
		submit(){
	    	var vm = this,
                params={
                    timeZone:timeZone,
                    operatorCode:operator_code
                },
                urls='';
			// 防止多次提交
            if(vmReboot.slideSubmitLoading)return

			if(vm.ruleForm.id){
				urls = '${ctx}/task/recurringReboot/updateTaskConfigInfo.action'
			}else{
				urls = '${ctx}/task/recurringReboot/saveTaskConfigInfo.action'
			}
			Object.assign(params,vm.ruleForm);

	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
                    vmReboot.slideSubmitLoading = true;
					axios.post(urls,stringify(params)).then(function(response){
						var data = response.data;
						if(data.success){
							vm.$message({
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							})
							eventBus.$emit('hide-recurringReboot-slide');
						}else{
							vm.$message.error(data["message"]);
                            vmReboot.slideSubmitLoading = false;
						}
		 			})
	    		}else{
	    			return false;
	    		}
	    	})
		},
		// 取消
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
			if(isFormChanged(this.$refs.ruleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('hide-recurringReboot-slide')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-recurringReboot-slide')
			}
		},
	
	},
	watch:{
		deviceSelection(){
			var data = this.$refs.devicePairgrid.getData();
			var selectedENB = '';
			if(data.length != 0){
				data.map(function(item){
					selectedENB += item.small_cell_code + ","
				})
			}
			this.ruleForm.selectedENB = selectedENB
		},
		versionSelection(){
            var data = this.$refs.versionPairgrid.getData();
			var selectedVersion = '',selectedVersionList=[];
			if(data.length != 0){
				data.map(function(item){
					selectedVersionList.push(item.software_version)
				})
			}
			selectedVersion = selectedVersionList.join(',')
			this.ruleForm.selectedVersion = selectedVersion;
        },
	},
	mounted(){
		eventBus.$off('recurringRebootTask-init').$on('recurringRebootTask-init',this.init);
		eventBus.$off('recurringRebootTask-submit').$on('recurringRebootTask-submit',this.submit);
		eventBus.$off('recurringRebootTask-cancel').$on('recurringRebootTask-cancel',this.cancel);
	}
})
</script>