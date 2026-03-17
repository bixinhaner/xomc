<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gsmBtsSettingPage{
	height: 100%;
	width: 100%;
}
#gsmBtsSettingPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gsmBtsSettingPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gsmBtsSettingPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gsmBtsSettingPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gsmBtsSettingPage .rightContentCls .contentTableTitle{
	display: flex;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gsmBtsSettingPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gsmBtsSettingPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gsmBtsSettingPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gsmBtsSettingPage .itemListBoxCls{
	padding-top: 5px;
}
#gsmBtsSettingPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gsmBtsSettingPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gsmBtsSettingPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gsmBtsSettingPage .el-form-item{
	margin-bottom: 20px;
}
#gsmBtsSettingPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gsmBtsSettingPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gsmBtsSettingPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
#gsmBtsSettingPage .rightContentCls .el-checkbox.is-bordered.el-checkbox--small{
	padding: 8px 15px 5px 10px;
}
#gsmBtsSettingPage .el-date-editor .el-range-input{
	font-size: 12px;
}
#gsmBtsSettingPage .el-date-editor .el-range__close-icon{
	font-size: 14px!important;
	line-height: 20px;
}
#gsmBtsSettingPage .nlSyncStatusBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
	margin-bottom: 10px;
}
#gsmBtsSettingPage .nlSyncStatusBoxCls >div{
	flex:1;
}
#gsmBtsSettingPage .paramItemLabel{
	color:#7a7992;
}
#gsmBtsSettingPage .paramItemValue{
	height: 18px;
}
</style>

<div id="gsmBtsSettingPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			BTS Setting
			<!-- 按钮  同步 -->
			<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSettingsClick" tip="<%=rb.getString("TongBu")%>">
				<span class="el-icon el-icon-circle-refresh"></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
					<el-collapse-item name="Sync">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Sync Settings</span>
							</p>
						</template>
						<div class="rightContentCls">
							<!--Sync Mode-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">Sync Mode</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='Sync_Mode' style="width:40%;min-width:500px;" label="Mode" label-width="160px">
										<el-select v-model='ruleForm.Sync_Mode'>
											<el-option label='FREE_Running' value='1'></el-option>
											<el-option label='GNSS' value='2'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
							<!--GNSS Sync-->
							<div v-show="ruleForm.Sync_Mode == '2'"> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">GNSS Sync</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='Sync_syncSource' style="width:100%;min-width:400px;" label="Sync Source" label-width="160px">
										<el-checkbox-group v-model="syncSourceSelectList" size="small">
											<el-checkbox v-for="(items,index) in syncSourceList" border :label="items.value" :key="items.value" @change="syncSourceItemChange(items)">{{items.label}}</el-checkbox>
										</el-checkbox-group>
									</el-form-item>
									<div style="display:flex;flex-wrap: wrap">
										<el-form-item label='Sync Status' style="width:40%;min-width:400px;" label-width="160px" >
											<el-se v-model='ruleForm.Sync_Status' :disabled="true"></el-input>
                                            <el-select v-model='ruleForm.Sync_Status' :disabled="true">
                                                <el-option label="<%= rb.getString("WeiTongBu")%>" value="0"></el-option>
                                                <el-option label="<%= rb.getString("TongBuChengGong")%>" value="1"></el-option>
                                                <el-option label="<%= rb.getString("ZhengZaiTongBu")%>" value="2"></el-option>
                                            </el-select>
										</el-form-item>
										<el-form-item label='Longitude' style="width:40%;min-width:400px;" label-width="160px" >
											<el-input v-model='ruleForm.Sync_Longitude' :disabled="true"></el-input>
										</el-form-item>
										<el-form-item label='Latitude' style="width:40%;min-width:400px;" label-width="160px" >
											<el-input v-model='ruleForm.Sync_Latitude' :disabled="true"></el-input>
										</el-form-item>
										<el-form-item label='Altitude' style="width:40%;min-width:400px;" label-width="160px" >
											<el-input v-model='ruleForm.Sync_Altitude' :disabled="true"></el-input>
										</el-form-item>
                                        <el-form-item label='Number Of Satellite' style="width:40%;min-width:400px;" label-width="160px" >
											<el-input v-model='ruleForm.Sync_NumberOfSatellite' :disabled="true"></el-input>
										</el-form-item>
									</div>
								</div>
							</div>
						</div>
					</el-collapse-item>
				</el-collapse>
			</el-form>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gsmBtsSettingPage = new Vue({
	el: '#gsmBtsSettingPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			},
			validateIPaddress= (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				
				if(value === ''){
					callback()
				}else{
					if(vm.isValidIP(value) || vm.isIPv6(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
					}
				}
			};
		return {
			activeCollapse:['CU','DU','Sync','Energy'],
			rowDataInfo: [],
			smallCellCode:'',
			syncSourceSelectList:[],
			syncSourceList:[{label:'GPS',value:'1'},{label:'GLONASS',value:'2'},{label:'BEIDOU',value:'8'},{label:'GALILEO',value:'16'},{label:'QZSS',value:'32'}],
			ruleForm:{
				Sync_Mode:'2',
				Sync_syncSource:'',
				Sync_Status:'',
				Sync_Longitude:'',
				Sync_Latitude:'',
				Sync_Altitude:'',
                Sync_NumberOfSatellite:''
			},
			rules:{},
			casts:{
				'9CCF35934288CA653B2DDA52CD72FF5A':'Sync_Mode',
				'F7D6D0831E3350CE7F36B38AFF528C97':'Sync_syncSource',
				'D29221D633B67AD2AB5731972182520C':'Sync_Status',
				'4B25D736D821C7806F1D054D84048804':'Sync_Altitude',
				'C1AD5A543AC9077B7B2C3FC1EA92890D':'Sync_Longitude',
				'F5AB8B8ECE7B5A20A8716E337C043494':'Sync_Latitude',
				'8439950355BB9A76E4512BC47AA5C223':'Sync_NumberOfSatellite',
			},
			optType:'',
			tbType:'',
		};
	},
	computed: {},
	watch: {},
	methods: {
		init(row){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = row.small_cell_code;

			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'40000');

		},
		getParamData(code,id) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
				params = {
					id: id,
					cellIndex:'1',
					smallCellCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				vm.resetFormData();
				if(data && Array.isArray(data)) {
					data.map(function(item){
						item.groups.map(function(group){
							group.list.map(function(m){
								if(m.type == 'list'){
									vm.initTable(m.url,m.label);
								}else{
									codes.push(m.name);
									// 执行赋值
									vm.setValue(m);
								}
							});
						});
					});
					initForm(vm.$refs.ruleForm);
				}
			});
		},
		// 映射赋值
		setValue(item) {
			var vm = this,
			code = item.name,
			value = item.value;

			// indexs是否含有
				var key = vm.casts[code];
			try{
				if(key){
					if(key == 'Sync_syncSource'){
						vm.syncSourceSelectList = value.split(',');
					}else{
                        vm.ruleForm[key] = value;
                    }
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					smallCellCode : vm.smallCellCode
			}
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							codes.push(key);
							obj[vm.casts[key]] = item[key]
						}
					})
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					Sync_Mode:'2',
					Sync_syncSource:'',
					Sync_syncStatus:'',
					Sync_Longitude:'',
					Sync_Latitude:'',
					Sync_Altitude:'',
					Sync_NumberOfSatellite:'',
				};
			Object.assign(vm.ruleForm,params);
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
		},
		// 验证输入的是否是整数
		isInteger(str) {
			if(str.length==0){
				return false;
			}
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(!reg.test(str)){
				return false;
			}
			return true;  
		},
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		settingsSubmit(){
			var vm = this;
			var params = {},
				isChanged = isFormChanged(vm.$refs.ruleForm),
				isSync = false;

			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}

			vm.$refs.ruleForm.fields.map(function(field){
				var key = vm.getNameByProp(field.prop);

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						var editList=[],subList=[];
						vList.map((items)=>{
							if(items.operateType){
								editList.push(items)
							}
						})
						editList.map((items)=>{
							if(items.operateType == 'add'){
								Object.keys(items).map((key)=>{
									if(key.slice(-3) == 'idx'){
										delete items[key]
									}
								})
							}
						})
						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								objs[listKey] = items[listVal]
							}
							objs.cellIndex = '1';
							subList.push(objs)
						})
						
						params[key] = subList;
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						params[key] = field.fieldValue;
					};
				}
			});
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
					var rowCode = vm.smallCellCode,
						url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
					$('#gsm_setting_main').addClass('loading');
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							vm.closeSettings();
						}else{
							vm.$message.error(data["message"])
						}
						$('#gsm_setting_main').removeClass('loading');
					})
				}
			});
		},
		getNameByProp(prop) {
			var vm = this,
				reg = /^\w*\.\d*\.\w*$/
				key = prop;
			
			if(reg.test(prop)) {
				var mReg = /\.(\d*)\./,
					sufReg = /\.(\w*)$/,
					idx = prop.match(mReg)[1],
					sufStr = prop.match(sufReg)[1];

				vm.codeList.map(function(name){
					var index = vm.indexs[name];
					if(vm.casts[name] == sufStr && index == idx) {
						key = name;
					}
				});
			}else {
				vm.codeList.map(function(name){
					if(vm.casts[name] == prop) {
						key = name;
					}
				});
			}

			return key;
		},
		createId(idVal,list){
			var vm = this,
				val = idVal + '';
			if(list.includes(val) == true){
				idVal += 1 ;
				return vm.createId(idVal,list);
			}else{
				return  idVal + '';
			}
		},
		closeSettings(){
			eventBus.$emit('gsm-close-setting');
		},
		// Sync_syncSource 点击事件
		syncSourceItemChange(clickItem){
			var vm = this;
				syncSourceSelectList = vm.syncSourceSelectList;
			if(clickItem.label == 'GLONASS'){
				vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
					return items != '8' && items != '16';
				})
			}else if(clickItem.label == 'BEIDOU'){
				vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
					return items != '2' && items != '16';
				})
			}else if(clickItem.label == 'GALILEO'){
				vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
					return items != '2' && items != '8';
				})
			}else if(clickItem.label == 'GPS'){
                let isExist = vm.syncSourceSelectList.some(items=>items == clickItem.value);
                if(!isExist){
                    vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
                        return items != '32';
                    })
                }
            }else if(clickItem.label == 'QZSS'){
                let isExist = vm.syncSourceSelectList.some(items=>items == clickItem.value);
                if(isExist){
                    let isExistGps = vm.syncSourceSelectList.some(items=>items == '1');
                    if(!isExistGps){
                        vm.syncSourceSelectList.push('1');
                    }
                }
            }
			vm.ruleForm.Sync_syncSource = vm.syncSourceSelectList.join(',');
		},
		//同步
		syncSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				params = {
					smallCellCode:vm.smallCellCode
				},
				str = Math.random().toString();
				
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					gsmSettingVue.changeMain('btsSetting');
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		//校验IP
        isValidIP(ip){
            var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
            return reg.test(ip);     
        },
        //Ipv6校验 
        isIPv6(str){ 
            var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
            return reg.test(str);
        },
	},
	mounted() {
		eventBus.$off("gsm-data").$on("gsm-data",this.init)
	}
});

</script>
