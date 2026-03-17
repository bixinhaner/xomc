<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#enbSettingLicensePage {
	display:flex;
	flex-direction:column;
	height:100%;
}
#enbSettingLicensePage .borderPage {
	border:1px solid #d5dcec;
	border-radius:10px;
	background:#fff;
	flex:1;
	height: 100%;
}
.licenceExpiredRemindCls {
	margin:-10px -10px 10px -10px;
	background:#fff;
	padding:5px 20px;
}
.cellItemBoxCls {
	display:flex;
	flex-wrap:wrap;
	padding:20px;
}
.cellItemBoxCls>div {
	width:40%;
	min-width:200px;
	min-height:40px;
}
.cellItemBoxCls>div:before {
	content:attr(label);
	display:inline-block;
	color:#7a7992;
	margin-bottom:5px;
	width:120px;
}
</style>

<div id="enbSettingLicensePage">
	<div class="licenceExpiredRemindCls" v-show="(delay_avaliable=='0' || delay_avaliable=='1') && writableMap['CODE_ENB_LICENSE'] == true">
		<div style="width:100%">
			<p><%=rb.getString("licenseGuoQiTiShi")%></p>
			<div style="margin-top:5px;">
			<el-button type="primary" :disabled="licDelayEnable" @click="extendLic"><%=rb.getString("JiuJi")%></el-button>
				<i class="el-icon el-icon-sas-warning" style="margin:0 5px;"></i>
				<span><%=rb.getString("licenseJiuJiJieShi")%></span>
			</div>
		</div>
	</div>
	<div class="borderPage"> 
		<div class="cellItemBoxCls">
			<div class="cell-item-cls" label="<%=rb.getString("LicenseBanBen")%>">{{licenceInfo.version}}</div>
			<div class="cell-item-cls" label="<%=rb.getString("ShengChengShiJian")%>">{{licenceInfo.generate_date}}</div>
			<div class="cell-item-cls" label="<%=rb.getString("LicenseMoShi")%>">{{licenceInfo.halob_mode}}</div>
		</div>
		<div style="height:calc(100% - 140px);border:1px solid #ddd;margin:0 20px;">
			<el-ctable ref="featureListTable" :rownumber="false" id="featureListTable" :data="featureListTableData" :pagination="false">
				<el-table-column label='<%=rb.getString("TeXingID")%>' width="90" prop="featureId"></el-table-column>
				<el-table-column label='<%=rb.getString("MiaoShu")%>' min-width="200" prop="description"></el-table-column>
				<el-table-column label='<%=rb.getString("ShuLiang")%>' min-width="120" prop="quantity"></el-table-column>
				<el-table-column label='<%=rb.getString("YouXiaoQi")%>' min-width="120" prop="validPeriod">
					<template slot-scope="scope">
						<div v-if="scope.row.validPeriod == '0'"><%=rb.getString("Yongjiu")%></div>
						<div v-else>{{scope.row.validPeriod}}</div>
					</template>
				</el-table-column>
				<!-- 如果有效期是0，剩余天数则显示永久； 反之 剩余天数则取值  remaining_period-->
				<el-table-column label='<%=rb.getString("ShengYuShiJian")%>' min-width="120" prop="remainingPeriod"></el-table-column> 
			</el-ctable> 
		</div>
		<%-- <el-ctable ref="featureListTable" :rownumber="false" id="featureListTable" :data="featureListTableData" style="border:1px solid #E9E9E9;" :pagination="false" :height="200px">
			<el-table-column label='<%=rb.getString("TeXingID")%>' width="90" prop="featureId"></el-table-column>
			<el-table-column label='<%=rb.getString("MiaoShu")%>' min-width="200" prop="description"></el-table-column>
			<el-table-column label='<%=rb.getString("ShuLiang")%>' min-width="120" prop="quantity"></el-table-column>
			<el-table-column label='<%=rb.getString("YouXiaoQi")%>' min-width="120" prop="validPeriod"></el-table-column>
			<el-table-column label='<%=rb.getString("ShengYuShiJian")%>' min-width="120" prop="remainingPeriod"></el-table-column>
		</el-ctable> --%>
	</div>

</div>

<script> 
var enbSettingLicenseVue = new Vue({
	el: '#enbSettingLicensePage',  
	data() {
		return {
            enbSelectedRow:{},
			licenceInfo:{
				serial_number:'',
				version:'',
				generate_date:'',
				halob_mode:'',
			},
			featureListTableData:[],
			delay_avaliable:'',
			licDelayEnable:true,
			code:''
		};
	},
	computed: {
		isCloud() {
			return isCloud == 'true';
		},
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		
	},
	watch:{
		
	},
	methods: {
        init(row,code,sn,status,version,product,licFlag){
			var vm = this;
            vm.enbSelectedRow = row;
			vm.code = code;
			vm.delay_avaliable = licFlag;
			if( licFlag == '1') vm.licDelayEnable = false;
			vm.getLicenceInfo();
		},
		// 获取licence信息
		getLicenceInfo(){
			var vm = this,
				urls = '${ctx}/cell/license/getHalobLicenseInfo.action',
				params = {
					small_cell_code: vm.code
				};
           
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				 if(!isEmptyObject(data)){
					 Object.assign(vm.licenceInfo,data);
					 var featureListTableData = [];
					 if(data.capacity_list){
						 data.capacity_list.map((item,index)=>{
							var obj={};
							obj.featureId = item.id;
							obj.validPeriod = item.valid_period;
							if (item.valid_period == "0"){
                                obj.remainingPeriod = '<%=rb.getString("Yongjiu")%>';
                            }
                            else{
                                obj.remainingPeriod = item.remaining_period;
                            }
							obj.quantity = item.capa_value;
                            obj.description = item.description;
                            featureListTableData.push(obj);
						 })
						 vm.featureListTableData = featureListTableData;
                    }
				 }
			}) 
		},
		// licence 救急
		extendLic(){
			var vm = this;
			
			vm.$confirm('<%=rb.getString("licenseYanQiQueRen")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal: false
	    	}).then(() => {
	    		var params = {smallCellCode: vm.code};
	    		vm.licDelayEnable = true;
				axios.post("${ctx}/cell/cpeinfos/goDelayEnbLic.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
							message: 'Success',
							type:'success',
						})
					}else {
						vm.$message({
							message: 'Fail',
							type:'error',
						})
						vm.licDelayEnable = false;
					}
				})
	    	}).catch(function(){})
		},
	},
	mounted() {
		eventBus.$off("enb-data").$on("enb-data",this.init)
	}
});

</script>
